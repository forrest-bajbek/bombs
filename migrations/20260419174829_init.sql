-- +goose Up
PRAGMA secure_delete = TRUE;

CREATE TABLE user (
    id INTEGER PRIMARY KEY
    , username TEXT NOT NULL UNIQUE COLLATE NOCASE
    , password_hash TEXT NOT NULL
    , is_admin BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE chat (
    id INTEGER PRIMARY KEY
    , name TEXT NOT NULL UNIQUE
);

CREATE TABLE chat_user (
    id INTEGER PRIMARY KEY
    , chat_id INTEGER NOT NULL REFERENCES chat(id)
    , user_id INTEGER NOT NULL REFERENCES user(id)
    , UNIQUE(chat_id, user_id)
);

-- +goose Down
DROP TABLE chat_user;
DROP TABLE chat;
DROP TABLE user;
