package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

func IsDatabaseAttached(db *sql.DB, dbName string) (bool, error) {
	rows, err := db.Query("PRAGMA database_list")
	if err != nil {
		return false, fmt.Errorf("failed to query database list: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return false, fmt.Errorf("failed to scan database list: %w", err)
		}
		if name == dbName {
			return true, nil
		}
	}
	return false, rows.Err()
}

type Repo struct {
	db        *sql.DB
	encrypter *utils.Encrypter
}

func NewRepo(db *sql.DB, encrypter *utils.Encrypter) *Repo {
	return &Repo{
		db:        db,
		encrypter: encrypter,
	}
}

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
	_, err := r.db.Exec("DELETE FROM user WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", userID, err)
	}
	return nil
}

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

func (r *Repo) CreateChat(requestingUserID int, chatName string) (int, error) {
	/*
		When Chat is created, requestingUserID is added as ChatUser in transaction.
		Doing this instead of a trigger because, in order to make that work, Chat
		would need to record a created_by_id, and I don't want to log that.
	*/
	if chatName == "" {
		return -1, errors.New("chatName cannot be empty")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	// Create Chat
	var chatID int
	err = tx.QueryRow("INSERT INTO chat (name) VALUES (?) RETURNING id", chatName).Scan(&chatID)
	if err != nil {
		return -1, err
	}

	// Add requestingUserID as a ChatUser
	_, err = tx.Exec("INSERT INTO chat_user (chat_id, user_id) VALUES (?, ?)", chatID, requestingUserID)
	if err != nil {
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return chatID, nil
}

func (r *Repo) IsUserInChat(userID int, chatID int) (bool, error) {
	var user_in_chat bool
	stmt := `
		SELECT COUNT(*) > 0 AS user_in_chat
		FROM user u
		INNER JOIN chat_user cu
			ON u.id = cu.user_id
		INNER JOIN chat c
			ON cu.chat_id = c.id
		WHERE
			u.id = ?
			AND c.id = ?
	`
	if err := r.db.QueryRow(stmt, userID, chatID).Scan(&user_in_chat); err != nil {
		return false, err
	}
	if !user_in_chat {
		return false, nil
	}
	return true, nil
}

func (r *Repo) UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only update chats of which you are a member.")
		return nil, err
	}

	var updatedChat types.Chat
	stmt := "UPDATE chat SET name = ? WHERE id = ? RETURNING id, name"
	err = r.db.QueryRow(stmt, chatName, chatID).Scan(&updatedChat.ID, &updatedChat.Name)
	if err != nil {
		return nil, err
	}
	return &updatedChat, nil
}

func (r *Repo) DeleteChat(requestingUserID int, chatID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		return errors.New("You can only delete chats of which you are a member.")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM message WHERE chat_id = ?", chatID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM chat_user WHERE chat_id = ?", chatID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM chat WHERE id = ?", chatID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *Repo) BombChat(requestingUserID int, chatID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		return errors.New("You can only bomb chats of which you are a member.")
	}

	_, err = r.db.Exec("DELETE FROM message WHERE chat_id = ?", chatID)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetChatByID(requestingUserID int, chatID int) (*types.Chat, error) {
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND ru.id = ?
		ORDER BY c.name
	`
	var chat types.Chat
	if err := r.db.QueryRow(stmt, chatID, requestingUserID).Scan(&chat.ID, &chat.Name); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *Repo) GetChat(requestingUserID int) (*[]types.Chat, error) {
	// Can only get chats to which you belong
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON cu.user_id = ru.id
		WHERE ru.id = ?
		ORDER BY c.name
	`
	rows, err := r.db.Query(stmt, requestingUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []types.Chat
	for rows.Next() {
		var chat types.Chat
		err := rows.Scan(&chat.ID, &chat.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chats = append(chats, chat)
	}
	return &chats, nil
}

func (r *Repo) SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error) {
	stmt := `
		SELECT c.id, c.name
		FROM chat c
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user u
			ON cu.user_id = ru.id
		WHERE
			INSTR(LOWER(c.name), ?) != 0
			AND u.id = ?
		ORDER BY c.name
	`
	rows, err := r.db.Query(stmt, strings.ToLower(chatName), requestingUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []types.Chat
	for rows.Next() {
		var chat types.Chat
		err := rows.Scan(&chat.ID, &chat.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chats = append(chats, chat)
	}
	return &chats, nil
}

// ChatUser
// ------------------------------------------------------------------------------------
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

func (r *Repo) DeleteChatUser(requestingUserID int, chatID int, userID int) error {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return err
	}
	if !user_in_chat {
		err := errors.New("You can only remove users from chats of which you are a member.")
		return err
	}

	if _, err := r.db.Exec("DELETE FROM chat_user WHERE chat_id = ? AND user_id = ?", chatID, userID); err != nil {
		return err
	}
	return nil
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
	if _, err := r.db.Exec("DELETE FROM chat_user WHERE user_id = ?", userID); err != nil {
		return err
	}
	return nil
}

func (r *Repo) DeleteChatUserByID(chatUserID int) error {
	if _, err := r.db.Exec("DELETE FROM chat_user WHERE id = ?", chatUserID); err != nil {
		return err
	}
	return nil
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

func (r *Repo) GetChatIDsForChannels() ([]int, error) {
	rows, err := r.db.Query("SELECT id FROM chat ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int
	for rows.Next() {
		var chatID int
		err := rows.Scan(&chatID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	if len(chatIDs) == 0 {
		return []int{}, nil
	}

	return chatIDs, nil
}

func (r *Repo) CreateMessage(requestingUserID int, chatID int, text string) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		err := errors.New("You can only create messages in chats of which you are a member.")
		return -1, err
	}

	// Encrypt text
	encryptedText, err := r.encrypter.Encrypt(text)
	if err != nil {
		return -1, err
	}

	// Create Chat
	var messageID int
	stmt := "INSERT INTO message (chat_id, user_id, encrypted_text) VALUES (?, ?, ?) RETURNING id"
	err = r.db.QueryRow(stmt, chatID, requestingUserID, encryptedText).Scan(&messageID)
	if err != nil {
		return -1, err
	}

	return messageID, nil
}

func (r *Repo) GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.ChannelMessage, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only retrieve messages for chats of which you are a member.")
		return nil, err
	}

	stmt := `
		SELECT
			m.id AS message_id
			, m.created_at AS message_created_at
			, c.id AS chat_id
			, mu.id AS user_id
			, mu.username
			, m.encrypted_text
		FROM message m
		INNER JOIN user mu
			ON m.user_id = mu.id
		INNER JOIN chat c
			ON m.chat_id = c.id
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND ru.id = ? -- requesting user is in the chat
		ORDER BY m.id
	`
	rows, err := r.db.Query(stmt, chatID, requestingUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []types.ChannelMessage
	for rows.Next() {
		var m types.ChannelMessage
		var encryptedText []byte
		err := rows.Scan(
			&m.MessageID,
			&m.MessageCreatedAt,
			&m.ChatID,
			&m.UserID,
			&m.Username,
			&encryptedText,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		text, err := r.encrypter.Decrypt(encryptedText)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt message: %w", err)
		}
		m.Text = text
		messages = append(messages, m)
	}

	if len(messages) == 0 {
		return &[]types.ChannelMessage{}, nil
	}
	return &messages, nil
}

func (r *Repo) GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.ChannelMessage, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return nil, err
	}
	if !user_in_chat {
		err := errors.New("You can only retrieve messages for chats of which you are a member.")
		return nil, err
	}

	stmt := `
		SELECT
			m.id AS message_id
			, m.created_at AS message_created_at
			, c.id AS chat_id
			, mu.id AS user_id
			, mu.username
			, m.encrypted_text
		FROM message m
		INNER JOIN user mu
			ON m.user_id = mu.id
		INNER JOIN chat c
			ON m.chat_id = c.id
		INNER JOIN chat_user cu
			ON c.id = cu.chat_id
		INNER JOIN user ru
			ON cu.user_id = ru.id
		WHERE
			c.id = ?
			AND m.id = ?
			AND ru.id = ? -- requesting user is in the chat
		ORDER BY m.id DESC
	`
	var m types.ChannelMessage
	var encryptedText []byte
	err = r.db.QueryRow(stmt, chatID, messageID, requestingUserID).Scan(
		&m.MessageID,
		&m.MessageCreatedAt,
		&m.ChatID,
		&m.UserID,
		&m.Username,
		&encryptedText,
	)
	if err != nil {
		return &m, err
	}
	text, err := r.encrypter.Decrypt(encryptedText)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}
	m.Text = text

	return &m, nil
}

func (r *Repo) GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error) {
	stmt := `
		WITH last_message AS (
			SELECT
				id
				, created_at
				, chat_id
				, user_id
				, encrypted_text
				, ROW_NUMBER() OVER (PARTITION BY chat_id ORDER BY id DESC) AS rn
			FROM message
		)
		SELECT
			c.id AS chat_id
			, c.name AS chat_name
			, COALESCE(u.username, '') AS last_message_username
			, COALESCE(lm.encrypted_text, '') AS last_message_text
			, COALESCE(lm.created_at, '') AS last_message
		FROM chat c
		INNER JOIN chat_user rcu
			ON c.id = rcu.chat_id
		INNER JOIN user ru
			ON rcu.user_id = ru.id
		LEFT OUTER JOIN last_message lm
			ON c.id = lm.chat_id
			AND lm.rn = 1
		LEFT OUTER JOIN user u
			ON lm.user_id = u.id
		WHERE ru.id = ? -- only chats in which requesting user is a member
		ORDER BY lm.id DESC, c.id DESC
	`
	rows, err := r.db.Query(stmt, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("Error running SQL: %w", err)
	}
	defer rows.Close()

	var chatPreviews []types.ChatPreview
	for rows.Next() {
		var p types.ChatPreview
		var encryptedText []byte
		err := rows.Scan(
			&p.ChatID,
			&p.ChatName,
			&p.LastMessageUsername,
			&encryptedText,
			&p.LastMessageCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan user row: %w", err)
		}
		text, err := r.encrypter.Decrypt(encryptedText)
		if err != nil {
			text = ""
		}
		p.LastMessageText = text
		chatPreviews = append(chatPreviews, p)
	}

	// Check for errors when iterating rows.
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating rows: %w", err)
	}

	// Return empty list if no rows.
	if chatPreviews == nil {
		chatPreviews = []types.ChatPreview{}
	}
	return &chatPreviews, nil
}

func (r *Repo) DeleteMessageByUserID(userID int) error {
	if _, err := r.db.Exec("DELETE FROM message WHERE user_id = ?", userID); err != nil {
		return err
	}
	return nil
}
