package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

func (r *Repo) CreateUser(username string, password string) (int, error) {
	password_hash := utils.HashPassword(password)
	stmt := `
		INSERT INTO user (username, password_hash)
		VALUES (?, ?)
		RETURNING id
	`
	var id int
	err := r.db.QueryRow(stmt, username, password_hash).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (r *Repo) UserExists(username string) (bool, error) {
	stmt := "SELECT COUNT(*) AS count_users FROM user WHERE username = ?"
	var count_users int
	err := r.db.QueryRow(stmt, username).Scan(&count_users)
	if err != nil {
		return false, fmt.Errorf("error retrieving user count for username %s: %w", username, err)
	}
	if count_users > 0 {
		return true, nil
	}
	return false, nil
}

func (r *Repo) GetUserByID(userID int) (*types.User, error) {
	stmt := "SELECT id, username, is_admin FROM user WHERE id = ?"
	var user = types.User{}
	if err := r.db.QueryRow(stmt, userID).Scan(&user.ID, &user.Username, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with id %d not found", userID)
		}
		return nil, fmt.Errorf("failed to get user by id %d: %w", userID, err)
	}
	return &user, nil
}

func (r *Repo) GetUserByUsername(username string) (*types.User, error) {
	stmt := "SELECT id, username, is_admin FROM user WHERE username = ?"
	var user = types.User{}
	if err := r.db.QueryRow(stmt, username).Scan(&user.ID, &user.Username, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("username %s not found", username)
		}
		return nil, fmt.Errorf("failed to get user by username %s: %w", username, err)
	}
	return &user, nil
}

func (r *Repo) GetUsers() (*[]types.User, error) {
	stmt := "SELECT id, username, is_admin FROM user ORDER BY username"
	rows, err := r.db.Query(stmt)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var users = []types.User{}

	for rows.Next() {
		u := types.User{}
		err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}
	return &users, nil
}

func (r *Repo) SearchUsers(username string) (*[]types.User, error) {
	stmt := `
		SELECT
			id
			, username
			, is_admin
		FROM user
		WHERE INSTR(LOWER(username), ?) != 0
		ORDER BY username
	`
	rows, err := r.db.Query(stmt, strings.ToLower(username))
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var users = []types.User{}

	for rows.Next() {
		u := types.User{}
		err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}
	return &users, nil
}

func (r *Repo) DeleteUser(userID int) error {
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
		`DELETE FROM chat_user 			WHERE user_id = ?`,
		`DELETE FROM user 				WHERE id = ?`,
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
