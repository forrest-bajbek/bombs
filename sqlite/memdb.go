package sqlite

import "database/sql"

// EnsureMemDB attaches the shared in-memory database that backs the
// message table and creates the table if it doesn't exist yet.
//
// This can't be handled as a normal goose migration: goose's
// migration-version bookkeeping lives in the persistent file database and
// survives restarts, but ":memory:" storage does not, so a tracked
// migration would only ever run once and be skipped on every later boot,
// leaving the table missing. EnsureMemDB must therefore run unconditionally
// on every startup, on the same connection the app holds pinned open for
// its lifetime (see main.go), so the attachment - and the table - stay
// alive for as long as the process runs.
func EnsureMemDB(db *sql.DB) error {
	if _, err := db.Exec("ATTACH DATABASE 'file:memdb?mode=memory&cache=shared' AS memdb"); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS memdb.message (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, encrypted_text BLOB NOT NULL
		)
	`)
	return err
}
