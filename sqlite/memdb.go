package sqlite

import "database/sql"

// EnsureMemDB attaches the shared in-memory database that backs the
// message and file tables and creates them if they don't exist yet.
//
// This can't be handled as a normal goose migration: goose's
// migration-version bookkeeping lives in the persistent file database and
// survives restarts, but ":memory:" storage does not, so a tracked
// migration would only ever run once and be skipped on every later boot,
// leaving the table missing. EnsureMemDB must therefore run unconditionally
// on every startup, on the same connection the app holds pinned open for
// its lifetime (see main.go), so the attachment - and the tables - stay
// alive for as long as the process runs.
//
// None of these tables declare foreign keys. That matches memdb.message,
// but it isn't only for consistency: SQLite requires a foreign key's parent
// table to live in the same database as the child, so file.chat_id can't
// reference chat(id) at all - chat is in the persistent file database. The
// driver also never enables PRAGMA foreign_keys, so the REFERENCES clauses
// in the goose migration aren't enforced either. Referential integrity is
// maintained by the transaction in Repo.CreateMessage and by the deletion
// paths in repo.go.
func EnsureMemDB(db *sql.DB) error {
	if _, err := db.Exec("ATTACH DATABASE 'file:memdb?mode=memory&cache=shared' AS memdb"); err != nil {
		return err
	}

	// Must run while memdb is still empty - SQLite refuses to change
	// auto_vacuum on a database that already has pages.
	//
	// This keeps the database image compact as chats are bombed, but it
	// does not shrink the process: the driver is WASM, and WASM linear
	// memory only ever grows. Freed pages are reused by later writes, so
	// memory settles at the high-water mark of live data rather than
	// climbing with every upload - measured flat across repeated
	// bomb-and-refill cycles. What actually bounds growth is
	// types.StorageBudgetBytes, checked in the service layer.
	if _, err := db.Exec("PRAGMA memdb.auto_vacuum = FULL"); err != nil {
		return err
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS memdb.message (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, encrypted_text BLOB NOT NULL
		)`,
		// chat_id and user_id are denormalized onto file so that authorizing
		// a download is a single join against chat_user, and so that the
		// bomb/delete paths don't have to walk message_file.
		//
		// There is deliberately no filename column: it's metadata we'd be
		// holding at rest for no benefit, and not storing it means there's
		// nothing to sanitize when setting response headers.
		//
		// size_bytes is the plaintext length. The ciphertext is a little
		// longer and not exactly invertible, so this is what the storage
		// budget check measures.
		`CREATE TABLE IF NOT EXISTS memdb.file (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			, chat_id INTEGER NOT NULL
			, user_id INTEGER NOT NULL
			, mime_type TEXT NOT NULL
			, size_bytes INTEGER NOT NULL
			, encrypted_content BLOB NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS memdb.message_file (
			id INTEGER PRIMARY KEY AUTOINCREMENT
			, message_id INTEGER NOT NULL
			, file_id INTEGER NOT NULL
			, position INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_file_chat_id ON file (chat_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_file_user_id ON file (user_id)`,
		`CREATE INDEX IF NOT EXISTS memdb.idx_message_file_message_id ON message_file (message_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
