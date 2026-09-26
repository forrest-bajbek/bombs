package sqlite

import (
	"errors"
	"log"
	"os"

	"github.com/forrest-bajbek/bombs/utils"
)

func (r *Repo) CheckPassword(username string, password string) (int, error) {
	stmt := "SELECT id, password_hash FROM user WHERE username = ?"
	var id int
	var password_hash string
	var errWrongUsernameOrPassword = errors.New("username or password is incorrect")
	if err := r.db.QueryRow(stmt, username).Scan(&id, &password_hash); err != nil {
		return -1, errWrongUsernameOrPassword
	}
	if !utils.CheckPasswordHash(password, password_hash) {
		return -1, errWrongUsernameOrPassword
	}
	return id, nil
}

func (r *Repo) ChangePassword(username string, old_password string, new_password string) error {
	user_id, err := r.CheckPassword(username, old_password)
	if err != nil {
		return err
	}

	new_password_hash := utils.HashPassword(new_password)
	stmt := "UPDATE user SET password_hash = ? WHERE id = ? RETURNING id"
	_, err = r.db.Exec(stmt, new_password_hash, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) EnsureAdmin() error {
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

	if !(usernameSpecified == passwordSpecified) {
		var errEnvVarNotSpecifiedAsPair = errors.New(
			"Environemnt Variables BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD must be specified together or not at all.",
		)
		return errEnvVarNotSpecifiedAsPair
	}

	// retrieve all usernames where is_admin
	rows, err := r.db.Query("SELECT username FROM user WHERE is_admin")
	if err != nil {
		log.Fatal(err)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	var admin_usernames []string
	for rows.Next() {
		var u string
		err := rows.Scan(&u)
		if err != nil {
			log.Fatal(err)
		}
		admin_usernames = append(admin_usernames, u)
	}
	admin_count := len(admin_usernames)

	switch admin_count {
	case 0:
		log.Println("No admin found. Creating user...")
		if !usernameSpecified && !passwordSpecified {
			log.Printf(
				"BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD not set. Using default values (username=admin, password=admin)...",
			)
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
			requires_update := false
			admin_username := admin_usernames[0]
			if admin_username != username {
				requires_update = true
			} else {
				_, err = r.CheckPassword(username, password)
				if err != nil {
					requires_update = true
				}
			}
			if requires_update {
				log.Println("Updating admin credentials to values specified in BOMBS_ADMIN_USERNAME and BOMBS_ADMIN_PASSWORD...")
				password_hash := utils.HashPassword(password)
				stmt := "UPDATE user SET username = ?, password_hash = ? WHERE is_admin"
				if _, err := r.db.Exec(stmt, username, password_hash); err != nil {
					return err
				}
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
