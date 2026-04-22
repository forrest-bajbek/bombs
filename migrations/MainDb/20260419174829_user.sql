-- +goose Up
CREATE TABLE user (
    id INTEGER PRIMARY KEY AUTOINCREMENT
    , created_at DATETIME NOT NULL DEFAULT (DATETIME('now'))
    , updated_at DATETIME NOT NULL DEFAULT (DATETIME('now'))
    , username VARCHAR(20) NOT NULL
    , password_hash VARCHAR(255) NOT NULL
    , is_admin BOOLEAN NOT NULL DEFAULT FALSE
);


-- +goose Down
DROP TABLE user;
