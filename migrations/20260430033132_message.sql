-- +goose Up
-- ATTACH DATABASE 'file:memdb?mode=memory&cache=shared' AS memdb;
CREATE TABLE message (
    id INTEGER PRIMARY KEY AUTOINCREMENT
    , created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    , chat_id INTEGER NOT NULL
    , user_id INTEGER NOT NULL
    , encrypted_text BLOB NOT NULL
);

-- +goose Down
DROP TABLE message;
-- DETACH DATABASE memdb;
