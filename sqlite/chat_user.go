package sqlite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/forrest-bajbek/bombs/types"
)

func (r *Repo) AddChatUser(requestingUserID int, chatID int, userID int) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		err := errors.New("You can only add users to chats of which you are a member.")
		return -1, err
	}

	var chatUserID int
	stmt := "INSERT INTO chat_user (chat_id, user_id) VALUES (?, ?) RETURNING id"
	err = r.db.QueryRow(stmt, chatID, userID).Scan(&chatUserID)
	if err != nil {
		return -1, err
	}
	return chatUserID, nil
}

func (r *Repo) GetChatUser(requestingUserID int, chatID int) (*[]types.User, error) {
	// Can only get users of chats to which requesting user belongs
	stmt := `
		SELECT u.id, u.username, u.is_admin
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON cu.user_id = u.id
		INNER JOIN chat_user rcu
			ON c.id = rcu.chat_id
		INNER JOIN user ru
			ON rcu.user_id = ru.id
		WHERE
			c.id = ?
			AND ru.id = ?
		ORDER BY u.username
	`
	rows, err := r.db.Query(stmt, chatID, requestingUserID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var user types.User
		err := rows.Scan(&user.ID, &user.Username, &user.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}
	return &users, nil
}

func (r *Repo) GetChatUserByUserID(requestingUserID int, chatID int, userID int) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		return -1, errors.New("You are not in this chat.")
	}

	stmt := `
		SELECT u.id AS chat_user_id
		FROM chat c
		INNER JOIN chat_user rcu
			ON c.id = rcu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		INNER JOIN user u
			ON cu.user_id = u.id
		WHERE
			c.id = ?
			AND u.id = ?
			AND ru.id = ?
		ORDER BY u.username
	`
	var chatUserID int
	err = r.db.QueryRow(stmt, chatID, userID, requestingUserID).Scan(&chatUserID)
	if err != nil {
		return -1, err
	}

	return chatUserID, nil
}

func (r *Repo) GetSuggestedUsers(requestingUserID int) (*[]types.User, error) {
	// Can only get users of chats to which requesting user belongs
	stmt := `
		WITH temp_linked_users AS (
			SELECT
				u.id AS user_id
				, COUNT(DISTINCT c.id) AS count_chat
			FROM user u
			INNER JOIN chat_user cu
				ON u.id = cu.user_id
			INNER JOIN chat c
				ON cu.chat_id = c.id
			INNER JOIN chat_user rcu
				ON c.id = rcu.chat_id
			INNER JOIN user ru
				ON rcu.user_id = ru.id
			WHERE ru.id = ?
			GROUP BY 1
		)
		SELECT u.id, u.username, u.is_admin
		FROM user u
		INNER JOIN temp_linked_users lu
			ON u.id = lu.user_id
		ORDER BY lu.count_chat DESC
	`
	rows, err := r.db.Query(stmt, requestingUserID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var user types.User
		err := rows.Scan(&user.ID, &user.Username, &user.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}
	return &users, nil
}

func (r *Repo) SearchForNewUsers(requestingUserID int, chatID int, search_term string) (*[]types.User, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		return nil, errors.New("You are not in this chat.")
	}

	if strings.TrimSpace(search_term) == "" {
		return &[]types.User{}, nil
	}

	// Can only get chats to which you belong
	stmt := `
		WITH temp_chat_user AS (
			SELECT u.id
			FROM chat c
			INNER JOIN chat_user cu
				ON c.id = cu.chat_id
			INNER JOIN user u
				ON cu.user_id = u.id
			WHERE c.id = ?
		)
		SELECT DISTINCT
			u.id
			, u.username
			, u.is_admin
		FROM user u
		LEFT OUTER JOIN temp_chat_user cu
			ON u.id = cu.id
		WHERE
			INSTR(LOWER(u.username), ?) > 0
			AND cu.id IS NULL -- new users can't already be in the chat
	`
	rows, err := r.db.Query(stmt, chatID, strings.ToLower(search_term))
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var user types.User
		err := rows.Scan(&user.ID, &user.Username, &user.IsAdmin)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	return &users, nil
}

func (r *Repo) DeleteChatUserByUserID(userID int) error {
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

func (r *Repo) DeleteChatUser(requestingUserID int, chatID int, userID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		err := errors.New("You can only remove users from chats of which you are a member.")
		return err
	}

	// start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// run statements
	stmts := []string{
		`DELETE FROM memdb.message_file WHERE chat_id = ? AND user_id = ?`,
		`DELETE FROM memdb.file 		WHERE chat_id = ? AND user_id = ?`,
		`DELETE FROM memdb.message 		WHERE chat_id = ? AND user_id = ?`,
		`DELETE FROM chat_user 			WHERE chat_id = ? AND user_id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, chatID, userID); err != nil {
			return err
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
