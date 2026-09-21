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

	if err := deleteChatFilesTx(tx, chatID); err != nil {
		return err
	}

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

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteChatFilesTx(tx, chatID); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM message WHERE chat_id = ?", chatID); err != nil {
		return err
	}

	return tx.Commit()
}

// deleteChatFilesTx removes every attachment belonging to a chat. The join
// rows go first so that no message_file row can ever outlive the file it
// points at. Attachments live in the in-memory database, which is only
// reclaimed on process restart, so any path that deletes messages has to
// delete their files too or the memory is gone for good.
func deleteChatFilesTx(tx *sql.Tx, chatID int) error {
	if _, err := tx.Exec(
		"DELETE FROM message_file WHERE file_id IN (SELECT id FROM file WHERE chat_id = ?)",
		chatID,
	); err != nil {
		return err
	}
	_, err := tx.Exec("DELETE FROM file WHERE chat_id = ?", chatID)
	return err
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
	if rows.Err() != nil {
		return nil, rows.Err()
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
	if rows.Err() != nil {
		return nil, rows.Err()
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

func (r *Repo) GetChatIDsForChannels() ([]int, error) {
	rows, err := r.db.Query("SELECT id FROM chat ORDER BY id")
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
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

func (r *Repo) CreateMessage(requestingUserID int, chatID int, text string, files []types.NewFile) (int, error) {
	user_in_chat, err := r.IsUserInChat(requestingUserID, chatID)
	if err != nil {
		return -1, err
	}
	if !user_in_chat {
		err := errors.New("You can only create messages in chats of which you are a member.")
		return -1, err
	}

	// Encrypt everything before opening the transaction. The pool is
	// pinned to a single connection (see main.go), so a transaction holds
	// the only connection the app has - running AES over several
	// megabytes of photos inside it would stall every other request,
	// including live message streams.
	encryptedText, err := r.encrypter.Encrypt(text)
	if err != nil {
		return -1, err
	}
	encryptedFiles := make([][]byte, len(files))
	for i, f := range files {
		encrypted, err := r.encrypter.Encrypt(string(f.Content))
		if err != nil {
			return -1, err
		}
		encryptedFiles[i] = encrypted
	}

	tx, err := r.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	var messageID int
	stmt := "INSERT INTO message (chat_id, user_id, encrypted_text) VALUES (?, ?, ?) RETURNING id"
	if err := tx.QueryRow(stmt, chatID, requestingUserID, encryptedText).Scan(&messageID); err != nil {
		return -1, err
	}

	for i, f := range files {
		var fileID int
		stmt := `
			INSERT INTO file (chat_id, user_id, mime_type, size_bytes, encrypted_content)
			VALUES (?, ?, ?, ?, ?)
			RETURNING id
		`
		if err := tx.QueryRow(
			stmt, chatID, requestingUserID, f.MimeType, len(f.Content), encryptedFiles[i],
		).Scan(&fileID); err != nil {
			return -1, err
		}

		stmt = "INSERT INTO message_file (message_id, file_id, position) VALUES (?, ?, ?)"
		if _, err := tx.Exec(stmt, messageID, fileID, i); err != nil {
			return -1, err
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return messageID, nil
}

// StorageUsedBytes reports the total plaintext size of every stored
// attachment, for the budget check in the service layer.
func (r *Repo) StorageUsedBytes() (int64, error) {
	var total int64
	err := r.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM file").Scan(&total)
	return total, err
}

// getMessageFiles loads attachment metadata for a chat, or for one message
// when messageID is non-zero.
//
// encrypted_content is deliberately absent from the select list: this runs
// over whole pages of history, and pulling the blobs would mean decrypting
// megabytes of photos just to render a grid of thumbnails. The bytes are
// fetched one at a time by GetFile instead.
func (r *Repo) getMessageFiles(chatID int, messageID int) (map[int][]types.MessageFile, error) {
	stmt := `
		SELECT
			mf.message_id
			, f.id
			, f.chat_id
			, f.mime_type
			, mf.position
		FROM message_file mf
		INNER JOIN file f
			ON mf.file_id = f.id
		INNER JOIN message m
			ON mf.message_id = m.id
		WHERE
			m.chat_id = ?
			AND (? = 0 OR mf.message_id = ?)
		ORDER BY mf.message_id, mf.position, mf.id
	`
	rows, err := r.db.Query(stmt, chatID, messageID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filesByMessage := map[int][]types.MessageFile{}
	for rows.Next() {
		var mID int
		var f types.MessageFile
		if err := rows.Scan(&mID, &f.FileID, &f.ChatID, &f.MimeType, &f.Position); err != nil {
			return nil, fmt.Errorf("failed to scan message_file row: %w", err)
		}
		filesByMessage[mID] = append(filesByMessage[mID], f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filesByMessage, nil
}

// GetFile returns one attachment's decrypted bytes.
//
// The join against chat_user is the authorization check: it establishes in
// a single query both that the file belongs to the chat named in the URL
// and that the requesting user is a member of that chat. A miss returns
// sql.ErrNoRows so the handler can answer 404 rather than 403, which keeps
// file ids from being enumerable.
func (r *Repo) GetFile(requestingUserID int, chatID int, fileID int) (*types.File, error) {
	stmt := `
		SELECT
			f.id
			, f.created_at
			, f.mime_type
			, f.encrypted_content
		FROM file f
		INNER JOIN chat_user cu
			ON f.chat_id = cu.chat_id
		WHERE
			f.id = ?
			AND f.chat_id = ?
			AND cu.user_id = ?
	`
	var file types.File
	var encryptedContent []byte
	if err := r.db.QueryRow(stmt, fileID, chatID, requestingUserID).Scan(
		&file.ID,
		&file.CreatedAt,
		&file.MimeType,
		&encryptedContent,
	); err != nil {
		return nil, err
	}

	content, err := r.encrypter.Decrypt(encryptedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file %d: %w", fileID, err)
	}
	file.Content = []byte(content)

	return &file, nil
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
	if rows.Err() != nil {
		return nil, rows.Err()
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

	filesByMessage, err := r.getMessageFiles(chatID, 0)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		messages[i].Files = filesByMessage[messages[i].MessageID]
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

	filesByMessage, err := r.getMessageFiles(chatID, messageID)
	if err != nil {
		return nil, err
	}
	m.Files = filesByMessage[messageID]

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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// file.user_id is the uploader, and a file is only ever attached to
	// its uploader's own message, so scoping by user_id covers exactly the
	// files whose messages are about to be deleted.
	if _, err := tx.Exec(
		"DELETE FROM message_file WHERE file_id IN (SELECT id FROM file WHERE user_id = ?)",
		userID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM file WHERE user_id = ?", userID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM message WHERE user_id = ?", userID); err != nil {
		return err
	}

	return tx.Commit()
}
