package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/forrest-bajbek/bombs/types"
)

func (r *Repo) CreateMessage(requestingUserID int, chatID int, text string, files []types.NewFile, replyToID int) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		err := errors.New("You can only create messages in chats of which you are a member.")
		return -1, err
	}
	var replyTo sql.NullInt64
	if replyToID > 0 {
		var exists bool
		stmt := "SELECT EXISTS(SELECT 1 FROM message WHERE id = ? AND chat_id = ?)"
		if err := r.db.QueryRow(stmt, replyToID, chatID).Scan(&exists); err != nil {
			return -1, err
		}
		if !exists {
			return -1, errors.New("You can only reply to messages in the same chat.")
		}
		replyTo = sql.NullInt64{Int64: int64(replyToID), Valid: true}
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
	stmt := "INSERT INTO message (chat_id, user_id, encrypted_text, reply_to_id) VALUES (?, ?, ?, ?) RETURNING id"
	if err := tx.QueryRow(stmt, chatID, requestingUserID, encryptedText, replyTo).Scan(&messageID); err != nil {
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
	m, replyToID, err := r.getMessage(requestingUserID, chatID, messageID)
	if err != nil {
		return m, err
	}
	if replyToID.Valid {
		// Parent could be gone after a bomb; render the reply without a preview then.
		parent, _, err := r.getMessage(requestingUserID, chatID, int(replyToID.Int64))
		if err == nil {
			m.ReplyTo = types.NewReplyPreview(parent)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return m, nil
}

// getMessage loads one message without resolving its reply preview.
func (r *Repo) getMessage(requestingUserID int, chatID int, messageID int) (*types.ChannelMessage, sql.NullInt64, error) {
	var replyToID sql.NullInt64
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, replyToID, err
	}
	if !user_in_chat {
		err := errors.New("You can only retrieve messages for chats of which you are a member.")
		return nil, replyToID, err
	}

	stmt := `
		SELECT
			m.id AS message_id
			, m.created_at AS message_created_at
			, c.id AS chat_id
			, mu.id AS user_id
			, mu.username
			, m.encrypted_text
			, m.reply_to_id
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
		&replyToID,
	)
	if err != nil {
		return &m, replyToID, err
	}
	text, err := r.encrypter.Decrypt(encryptedText)
	if err != nil {
		return nil, replyToID, fmt.Errorf("failed to decrypt message: %w", err)
	}
	m.Text = text

	files, err := r.GetMessageFilesByMessageID(chatID, messageID)
	if err != nil {
		return nil, replyToID, err
	}
	m.Files = files

	return &m, replyToID, nil
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
			, m.reply_to_id
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
	var replyToIDs []sql.NullInt64
	for rows.Next() {
		var m types.ChannelMessage
		var encryptedText []byte
		var replyToID sql.NullInt64
		err := rows.Scan(
			&m.MessageID,
			&m.MessageCreatedAt,
			&m.ChatID,
			&m.UserID,
			&m.Username,
			&encryptedText,
			&replyToID,
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
		replyToIDs = append(replyToIDs, replyToID)
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
	byID := make(map[int]*types.ChannelMessage, len(messages))
	for i := range messages {
		byID[messages[i].MessageID] = &messages[i]
	}
	for i := range messages {
		if replyToIDs[i].Valid {
			if parent, ok := byID[int(replyToIDs[i].Int64)]; ok {
				messages[i].ReplyTo = types.NewReplyPreview(parent)
			}
		}
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

// DeleteMessage replaces the text of a message with "(deleted)" and removes its
// files. The row is kept so replies still point at it. Only the sender may delete.
func (r *Repo) DeleteMessage(requestingUserID int, chatID int, messageID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		return errors.New("You can only delete messages in chats of which you are a member.")
	}
	encryptedText, err := r.encrypter.Encrypt("(deleted)")
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var owned bool
	stmt := "SELECT EXISTS(SELECT 1 FROM memdb.message WHERE id = ? AND chat_id = ? AND user_id = ?)"
	if err := tx.QueryRow(stmt, messageID, chatID, requestingUserID).Scan(&owned); err != nil {
		return err
	}
	if !owned {
		return sql.ErrNoRows
	}

	if _, err := tx.Exec(`DELETE FROM memdb.message_file WHERE message_id = ?`, messageID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM memdb.file WHERE message_id = ?`, messageID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE memdb.message SET encrypted_text = ? WHERE id = ?`, encryptedText, messageID); err != nil {
		return err
	}

	return tx.Commit()
}
