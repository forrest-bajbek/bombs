package sqlite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/forrest-bajbek/bombs/types"
)

func (r *Repo) CreateChat(requestingUserID int, chatName string) (int, error) {
	/*
		When Chat is created, requestingUserID is added as ChatUser in transaction.
		Doing this instead of a trigger because, in order to make that work, Chat
		would need to record a created_by_id, and I don't want to log that.
	*/
	if chatName == "" {
		return -1, errors.New("chatName cannot be empty")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	// Create Chat
	var chatID int
	err = tx.QueryRow("INSERT INTO chat (name) VALUES (?) RETURNING id", chatName).Scan(&chatID)
	if err != nil {
		return -1, err
	}

	// Add requestingUserID as a ChatUser
	_, err = tx.Exec("INSERT INTO chat_user (chat_id, user_id) VALUES (?, ?)", chatID, requestingUserID)
	if err != nil {
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return chatID, nil
}

func (r *Repo) IsUserInChat(userID int, chatID int) (bool, error) {
	var user_in_chat bool
	stmt := `
		SELECT COUNT(*) > 0 AS user_in_chat
		FROM user u
		INNER JOIN chat_user cu
			ON u.id = cu.user_id
		INNER JOIN chat c
			ON cu.chat_id = c.id
		WHERE
			u.id = ?
			AND c.id = ?
	`
	if err := r.db.QueryRow(stmt, userID, chatID).Scan(&user_in_chat); err != nil {
		return false, err
	}
	if !user_in_chat {
		return false, nil
	}
	return true, nil
}

func (r *Repo) GetChatIDsForChannels() ([]int, error) {
	rows, err := r.db.Query("SELECT id FROM chat ORDER BY id")
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var chatIDs []int
	for rows.Next() {
		var chatID int
		err := rows.Scan(&chatID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	if len(chatIDs) == 0 {
		return []int{}, nil
	}

	return chatIDs, nil
}

func (r *Repo) GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error) {
	stmt := `
		WITH last_message AS (
			SELECT
				id
				, created_at
				, chat_id
				, user_id
				, encrypted_text
				, ROW_NUMBER() OVER (PARTITION BY chat_id ORDER BY id DESC) AS rn
			FROM message
		)
		SELECT
			c.id AS chat_id
			, c.name AS chat_name
			, COALESCE(u.username, '') AS last_message_username
			, COALESCE(lm.encrypted_text, '') AS last_message_text
			, COALESCE(lm.created_at, '') AS last_message_created_at
			, COALESCE(COUNT(f.id), 0) AS last_message_count_photos
		FROM chat c
		INNER JOIN chat_user rcu
			ON c.id = rcu.chat_id
		INNER JOIN user ru
			ON rcu.user_id = ru.id
		LEFT OUTER JOIN last_message lm
			ON c.id = lm.chat_id
			AND lm.rn = 1
		LEFT OUTER JOIN memdb.message_file mf
			ON lm.id = mf.message_id
		LEFT OUTER JOIN memdb.file f
			ON mf.file_id = f.id
		LEFT OUTER JOIN user u
			ON lm.user_id = u.id
		WHERE ru.id = ? -- only chats in which requesting user is a member
		GROUP BY 1, 2, 3, 4, 5
		ORDER BY lm.id DESC, c.id DESC
	`
	rows, err := r.db.Query(stmt, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("Error running SQL: %w", err)
	}
	defer rows.Close()

	var chatPreviews []types.ChatPreview
	for rows.Next() {
		var p types.ChatPreview
		var encryptedText []byte
		err := rows.Scan(
			&p.ChatID,
			&p.ChatName,
			&p.LastMessageUsername,
			&encryptedText,
			&p.LastMessageCreatedAt,
			&p.LastMessageCountPhotos,
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan user row: %w", err)
		}
		text, err := r.encrypter.Decrypt(encryptedText)
		if err != nil {
			text = ""
		}
		p.LastMessageText = text
		chatPreviews = append(chatPreviews, p)
	}

	// Check for errors when iterating rows.
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating rows: %w", err)
	}

	// Return empty list if no rows.
	if chatPreviews == nil {
		chatPreviews = []types.ChatPreview{}
	}
	return &chatPreviews, nil
}

func (r *Repo) UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only update chats of which you are a member.")
		return nil, err
	}

	var updatedChat types.Chat
	stmt := "UPDATE chat SET name = ? WHERE id = ? RETURNING id, name"
	err = r.db.QueryRow(stmt, chatName, chatID).Scan(&updatedChat.ID, &updatedChat.Name)
	if err != nil {
		return nil, err
	}
	return &updatedChat, nil
}

func (r *Repo) GetChatByID(requestingUserID int, chatID int) (*types.Chat, error) {
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND ru.id = ?
		ORDER BY c.name
	`
	var chat types.Chat
	if err := r.db.QueryRow(stmt, chatID, requestingUserID).Scan(&chat.ID, &chat.Name); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *Repo) GetChat(requestingUserID int) (*[]types.Chat, error) {
	// Can only get chats to which you belong
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON cu.user_id = ru.id
		WHERE ru.id = ?
		ORDER BY c.name
	`
	rows, err := r.db.Query(stmt, requestingUserID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var chats []types.Chat
	for rows.Next() {
		var chat types.Chat
		err := rows.Scan(&chat.ID, &chat.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chats = append(chats, chat)
	}
	return &chats, nil
}

func (r *Repo) SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error) {
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON cu.user_id = ru.id
		WHERE
			INSTR(LOWER(c.name), ?) != 0
			AND u.id = ?
		ORDER BY c.name
	`
	rows, err := r.db.Query(stmt, strings.ToLower(chatName), requestingUserID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var chats []types.Chat
	for rows.Next() {
		var chat types.Chat
		err := rows.Scan(&chat.ID, &chat.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chats = append(chats, chat)
	}
	return &chats, nil
}

func (r *Repo) BombChat(requestingUserID int, chatID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		return errors.New("You can only bomb chats of which you are a member.")
	}

	// start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// run statements
	stmts := []string{
		`DELETE FROM memdb.message_file WHERE chat_id = ?`,
		`DELETE FROM memdb.file 		WHERE chat_id = ?`,
		`DELETE FROM memdb.message 		WHERE chat_id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, chatID); err != nil {
			return err
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *Repo) DeleteChat(requestingUserID int, chatID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		return errors.New("You can only delete chats of which you are a member.")
	}

	// start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// run statements
	stmts := []string{
		`DELETE FROM memdb.message_file WHERE chat_id = ?`,
		`DELETE FROM memdb.file 		WHERE chat_id = ?`,
		`DELETE FROM memdb.message 		WHERE chat_id = ?`,
		`DELETE FROM chat_user 			WHERE chat_id = ?`,
		`DELETE FROM chat 				WHERE id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, chatID); err != nil {
			return err
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
