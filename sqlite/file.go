package sqlite

import (
	"fmt"

	"github.com/forrest-bajbek/bombs/types"
)

// For the budget check in the service layer.
func (r *Repo) StorageUsedBytes() (int64, error) {
	var total int64
	err := r.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM file").Scan(&total)
	return total, err
}

// getMessageFiles loads attachment metadata for a chat, or for one message
// when messageID is non-zero.

// encrypted_content is deliberately absent from the select list: this runs
// over whole pages of history, and pulling the blobs would mean decrypting
// megabytes of photos just to render a grid of thumbnails. The bytes are
// fetched one at a time by GetFile instead.
func (r *Repo) GetMessageFiles(chatID int, messageID int) (map[int][]types.MessageFile, error) {
	stmt := `
		SELECT
			mf.message_id
			, f.id AS file_id
			, f.chat_id
			, f.mime_type
			, mf.position
		FROM message_file mf
		INNER JOIN file f
			ON mf.file_id = f.id
		INNER JOIN message m
			ON mf.message_id = m.id
		WHERE
			m.chat_id = ?
			AND mf.message_id = ?
		ORDER BY mf.message_id, mf.position, mf.id
	`
	rows, err := r.db.Query(stmt, chatID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filesByMessage := map[int][]types.MessageFile{}
	for rows.Next() {
		var mID int
		var f types.MessageFile
		if err := rows.Scan(&mID, &f.FileID, &f.ChatID, &f.MimeType, &f.Position); err != nil {
			return nil, fmt.Errorf("failed to scan message_file row: %w", err)
		}
		filesByMessage[mID] = append(filesByMessage[mID], f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filesByMessage, nil
}

// GetFile returns one attachment's decrypted bytes.
func (r *Repo) GetFile(requestingUserID int, chatID int, fileID int) (*types.File, error) {
	stmt := `
		SELECT
			f.id
			, f.created_at
			, f.mime_type
			, f.encrypted_content
		FROM memdb.file f
		INNER JOIN memdb.message_file mf
			ON f.id = mf.file_id
		INNER JOIN memdb.message m
			ON mf.message_id = m.id
		INNER JOIN chat c
			ON m.chat_id = c.id
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON u.id = cu.user_id
		WHERE
			f.id = ?
			AND c.id = ?
			AND u.id = ?
	`
	var file types.File
	var encryptedContent []byte
	if err := r.db.QueryRow(stmt, fileID, chatID, requestingUserID).Scan(
		&file.ID,
		&file.CreatedAt,
		&file.MimeType,
		&encryptedContent,
	); err != nil {
		return nil, err
	}

	content, err := r.encrypter.Decrypt(encryptedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file %d: %w", fileID, err)
	}
	file.Content = []byte(content)

	return &file, nil
}
