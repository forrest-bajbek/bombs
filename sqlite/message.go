package sqlite

import (
	"errors"
	"fmt"

	"github.com/forrest-bajbek/bombs/types"
)

func (r *Repo) CreateMessage(requestingUserID int, chatID int, text string, files []types.NewFile) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		err := errors.New("You can only create messages in chats of which you are a member.")
		return -1, err
	}
	encryptedText, err := r.encrypter.Encrypt(text)
	if err != nil {
		return -1, err
	}
	encryptedFiles := make([][]byte, len(files))
	for i, f := range files {
		encrypted, err := r.encrypter.Encrypt(string(f.Content))
		if err != nil {
			return -1, err
		}
		encryptedFiles[i] = encrypted
	}

	tx, err := r.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	var messageID int
	stmt := "INSERT INTO message (chat_id, user_id, encrypted_text) VALUES (?, ?, ?) RETURNING id"
	if err := tx.QueryRow(stmt, chatID, requestingUserID, encryptedText).Scan(&messageID); err != nil {
		return -1, err
	}

	for i, f := range files {
		var fileID int
		stmt := `
			INSERT INTO file (message_id, chat_id, user_id, mime_type, size_bytes, encrypted_content)
			VALUES (?, ?, ?, ?, ?, ?)
			RETURNING id
		`
		if err := tx.QueryRow(
			stmt, messageID, chatID, requestingUserID, f.MimeType, len(f.Content), encryptedFiles[i],
		).Scan(&fileID); err != nil {
			return -1, err
		}

		stmt = "INSERT INTO message_file (message_id, file_id, chat_id, user_id, position) VALUES (?, ?, ?, ?, ?)"
		if _, err := tx.Exec(stmt, messageID, fileID, chatID, requestingUserID, i); err != nil {
			return -1, err
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return messageID, nil
}

func (r *Repo) GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.ChannelMessage, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only retrieve messages for chats of which you are a member.")
		return nil, err
	}

	stmt := `
		SELECT
			m.id AS message_id
			, m.created_at AS message_created_at
			, c.id AS chat_id
			, mu.id AS user_id
			, mu.username
			, m.encrypted_text
		FROM message m
		INNER JOIN user mu
			ON m.user_id = mu.id
		INNER JOIN chat c
			ON m.chat_id = c.id
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND m.id = ?
			AND ru.id = ? -- requesting user is in the chat
		ORDER BY m.id DESC
	`
	var m types.ChannelMessage
	var encryptedText []byte
	err = r.db.QueryRow(stmt, chatID, messageID, requestingUserID).Scan(
		&m.MessageID,
		&m.MessageCreatedAt,
		&m.ChatID,
		&m.UserID,
		&m.Username,
		&encryptedText,
	)
	if err != nil {
		return &m, err
	}
	text, err := r.encrypter.Decrypt(encryptedText)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}
	m.Text = text

	files, err := r.GetMessageFilesByMessageID(chatID, messageID)
	if err != nil {
		return nil, err
	}
	m.Files = files

	return &m, nil
}

func (r *Repo) GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.ChannelMessage, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only retrieve messages for chats of which you are a member.")
		return nil, err
	}

	stmt := `
		SELECT
			m.id AS message_id
			, m.created_at AS message_created_at
			, c.id AS chat_id
			, mu.id AS user_id
			, mu.username
			, m.encrypted_text
		FROM message m
		INNER JOIN user mu
			ON m.user_id = mu.id
		INNER JOIN chat c
			ON m.chat_id = c.id
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND ru.id = ? -- requesting user is in the chat
		ORDER BY m.id
	`
	rows, err := r.db.Query(stmt, chatID, requestingUserID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var messages []types.ChannelMessage
	for rows.Next() {
		var m types.ChannelMessage
		var encryptedText []byte
		err := rows.Scan(
			&m.MessageID,
			&m.MessageCreatedAt,
			&m.ChatID,
			&m.UserID,
			&m.Username,
			&encryptedText,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		text, err := r.encrypter.Decrypt(encryptedText)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt message: %w", err)
		}
		m.Text = text
		messages = append(messages, m)
	}

	if len(messages) == 0 {
		return &[]types.ChannelMessage{}, nil
	}

	filesByMessage, err := r.GetMessageFilesByChatID(chatID)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		messages[i].Files = filesByMessage[messages[i].MessageID]
	}

	return &messages, nil
}

func (r *Repo) DeleteMessageByUserID(userID int) error {
	// start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// run statements
	stmts := []string{
		`DELETE FROM memdb.message_file WHERE user_id = ?`,
		`DELETE FROM memdb.file 		WHERE user_id = ?`,
		`DELETE FROM memdb.message 		WHERE user_id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, userID); err != nil {
			return err
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
