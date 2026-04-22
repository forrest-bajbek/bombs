package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/forrest-bajbek/bombs/user"
	"github.com/forrest-bajbek/bombs/utils"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
	stmt := "SELECT id, created_at, updated_at, username, is_admin FROM user ORDER BY username"
	rows, err := r.db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users = []user.User{}
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Username, &u.IsAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) CreateUser(username string, password string) (int, error) {
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

func (r *UserRepository) GetUserByID(id int) (*user.User, error) {
	stmt := "SELECT id, created_at, updated_at, username, is_admin FROM user WHERE id = ?"
	var user = user.User{}
	if err := r.db.QueryRow(stmt, id).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Username, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id %d: %w", id, err)
	}
	return &user, nil
}

func (r *UserRepository) GetUsers() (*[]user.User, error) {
	stmt := "SELECT id, created_at, updated_at, username, is_admin FROM user ORDER BY username"
	rows, err := r.db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users = []user.User{}

	for rows.Next() {
		u := user.User{}
		err := rows.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Username, &u.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}
	return &users, nil
}

func (r *UserRepository) GetUsersByUsername(username string) (*[]user.User, error) {
	stmt := `
		SELECT
			id
			, created_at
			, updated_at
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
	defer rows.Close()

	var users = []user.User{}

	for rows.Next() {
		u := user.User{}
		err := rows.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Username, &u.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}
	return &users, nil
}

func (r *UserRepository) DeleteUserByID(id int) error {
	_, err := r.db.Exec("DELETE FROM user WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user with id %d not found", id)
		}
		return fmt.Errorf("failed to get user by id %d: %w", id, err)
	}
	return nil
}

func (r *UserRepository) CheckPassword(username string, password string) (int, error) {
	stmt := "SELECT id, password_hash FROM user WHERE username = ?"
	var id int
	var password_hash string
	var errWrongUsernameOrPassword = errors.New("username or password is incorrect")
	err := r.db.QueryRow(stmt, username).Scan(&id, &password_hash)
	if err != nil {
		return -1, err
	}
	if !utils.CheckPasswordHash(password, password_hash) {
		return -1, errWrongUsernameOrPassword
	}
	return id, nil
}

func (r *UserRepository) ChangePassword(username string, old_password string, new_password string) error {
	user_id, err := r.CheckPassword(username, old_password)
	if err != nil {
		return err
	}

	new_password_hash := utils.HashPassword(new_password)
	stmt := `
		UPDATE user
		SET
			password_hash = ?
			, updated_at = datetime('now')
		WHERE user_id = ?
		RETURNING id
	`
	_, err = r.db.Exec(stmt, new_password_hash, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) EnsureAdmin() error {
	/*
		Get count of admins
			- if count = 0, create admin
			- if count = 1 and creds specified in env vars, update the username/password
			- if count = 1 and creds not specified, do nothing
			- if count > 1, panic
	*/
	username := os.Getenv("BOMBS_ADMIN_USERNAME")
	password := os.Getenv("BOMBS_ADMIN_PASSWORD")
	usernameSpecified := username != ""
	passwordSpecified := password != ""

	if (usernameSpecified && !passwordSpecified) || (!usernameSpecified && passwordSpecified) {
		var errEnvVarNotSpecifiedAsPair = errors.New("Environemnt Variables BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD must be specified together or not at all.")
		return errEnvVarNotSpecifiedAsPair
	}

	stmt := "SELECT COALESCE(COUNT(*), 0) AS adminCount FROM user WHERE is_admin"
	var adminCount int
	if err := r.db.QueryRow(stmt).Scan(&adminCount); err != nil {
		return err
	}

	switch adminCount {
	case 0:
		log.Println("Creating admin account...")
		if !usernameSpecified && !passwordSpecified {
			log.Printf("BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD not set. Using default values...")
			username = "admin"
			password = "admin"
		}
		stmt := "INSERT INTO user (username, password_hash, is_admin) VALUES (?, ?, TRUE)"
		password_hash := utils.HashPassword(password)
		if _, err := r.db.Exec(stmt, username, password_hash); err != nil {
			return err
		}
		return nil
	case 1:
		if usernameSpecified && passwordSpecified {
			log.Println("Setting admin credentials to values specified in BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD...")
			password_hash := utils.HashPassword(password)
			stmt := "UPDATE user SET username = ?, password_hash = ? WHERE is_admin"
			if _, err := r.db.Exec(stmt, username, password_hash); err != nil {
				return err
			}
			return nil
		} else {
			return nil
		}
	default:
		var errMoreThanOneAdmin = errors.New("More than 1 admin account found. The system is not designed to handle this. If you didn't do this, it's possible you were hacked.")
		return errMoreThanOneAdmin
	}
}
