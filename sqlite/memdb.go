package sqlite

import "database/sql"

func EnsureMemDB(db *sql.DB) error {
	stmts := []string{
		`ATTACH DATABASE 'file:memdb?mode=memory&cache=shared' AS memdb`,
		`PRAGMA memdb.auto_vacuum = FULL`,
		`CREATE TABLE IF NOT EXISTS memdb.message (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, encrypted_text BLOB NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS memdb.file (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			, message_id INTEGER NOT NULL
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, mime_type TEXT NOT NULL
			, size_bytes INTEGER NOT NULL
			, encrypted_content BLOB NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS memdb.message_file (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, message_id INTEGER NOT NULL
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, file_id INTEGER NOT NULL
			, position INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_messsage_user_id ON message (user_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_file_chat_id ON file (chat_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_file_user_id ON file (user_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_message_file_chat_id ON message_file (chat_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_message_file_user_id ON message_file (user_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_message_file_message_id ON message_file (message_id)`,
		// Unfortunately can't use triggers against user, chat, and chat_user that operate on these tables. Handled in functions
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
