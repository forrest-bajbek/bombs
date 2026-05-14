package services

import (
	"errors"

	"github.com/forrest-bajbek/bombs/types"
)

type Repo interface {
	CreateUser(username string, password string) (int, error)
	CheckPassword(username string, password string) (int, error)
	ChangePassword(username string, old_password string, new_password string) error
	UserExists(username string) (bool, error)
	GetUserByID(userID int) (*types.User, error)
	GetUserByUsername(username string) (*types.User, error)
	GetUsers() (*[]types.User, error)
	SearchUsers(username string) (*[]types.User, error)
	DeleteUser(userID int) error
	EnsureAdmin() error

	CreateChat(requestingUserID int, chatName string) (int, error)
	UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error)
	IsUserInChat(userID int, chatID int) (bool, error)
	DeleteChat(requestingUserID int, chatID int) error
	GetChatByID(requestingUserID int, chatID int) (*types.Chat, error)
	GetChat(requestingUserID int) (*[]types.Chat, error)
	GetSuggestedUsers(requestingUserID int) (*[]types.User, error)
	SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error)
	AddChatUser(requestingUserID int, chatID int, userID int) (int, error)
	DeleteChatUser(requestingUserID int, chatID int, userID int) error
	GetChatUser(requestingUserID int, chatID int) (*[]types.User, error)
	DeleteChatUserByUserID(userID int) error
	GetChatUserByUserID(requestingUserID int, chatID int, userID int) (int, error)
	SearchForNewUsers(requestingUserID int, chatID int, search_term string) (*[]types.User, error)
	GetChatIDsForChannels() ([]int, error)

	CreateMessage(requestingUserID int, chatID int, text string) (int, error)
	GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.MessageDetail, error)
	GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.MessageDetail, error)
	GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error)
	DeleteMessageByUserID(userID int) error
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// User
func (s *Service) CreateUser(username string, password string) (int, error) {
	if len(username) < 2 || len(username) > 24 {
		return -1, errors.New("Username must be betwee 2 and 24 characters.")
	}
	if len(password) < 6 || len(password) > 64 {
		return -1, errors.New("Password must be between 6 and 64 characters.")
	}
	return s.repo.CreateUser(username, password)
}

func (s *Service) CheckPassword(username string, password string) (int, error) {
	if len(username) < 2 || len(username) > 24 || len(password) < 6 || len(password) > 64 {
		return -1, errors.New("username or password is incorrect")
	}
	return s.repo.CheckPassword(username, password)
}

func (s *Service) ChangePassword(username string, old_password string, new_password string) error {
	if len(new_password) < 6 || len(new_password) > 64 {
		return errors.New("New password must be between 6 and 64 characters.")
	}
	return s.repo.ChangePassword(username, old_password, new_password)
}

func (s *Service) UserExists(username string) (bool, error) {
	if len(username) < 2 || len(username) > 24 {
		return false, errors.New("Usernames must be between 2 and 24 characters long.")
	}
	return s.repo.UserExists(username)
}

func (s *Service) GetUserByID(userID int) (*types.User, error) {
	return s.repo.GetUserByID(userID)
}
func (s *Service) GetUserByUsername(username string) (*types.User, error) {
	return s.repo.GetUserByUsername(username)
}
func (s *Service) GetUsers() (*[]types.User, error) {
	return s.repo.GetUsers()
}
func (s *Service) SearchUsers(username string) (*[]types.User, error) {
	return s.repo.SearchUsers(username)
}
func (s *Service) DeleteUser(userID int) error {
	return s.repo.DeleteUser(userID)
}
func (s *Service) EnsureAdmin() error {
	return s.repo.EnsureAdmin()
}

// Chat
func (s *Service) CreateChat(requestingUserID int, chatName string) (int, error) {
	if len(chatName) < 1 || len(chatName) > 32 {
		return -1, errors.New("Chat name must be between 1 and 32 characters.")
	}
	return s.repo.CreateChat(requestingUserID, chatName)
}

func (s *Service) UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error) {
	if len(chatName) < 1 || len(chatName) > 32 {
		return nil, errors.New("Chat name must be between 1 and 32 characters.")
	}
	return s.repo.UpdateChat(requestingUserID, chatID, chatName)
}

func (s *Service) IsUserInChat(userID int, chatID int) (bool, error) {
	return s.repo.IsUserInChat(userID, chatID)
}
func (s *Service) DeleteChat(requestingUserID int, chatID int) error {
	return s.repo.DeleteChat(requestingUserID, chatID)
}
func (s *Service) GetChatByID(requestingUserID int, chatID int) (*types.Chat, error) {
	return s.repo.GetChatByID(requestingUserID, chatID)
}
func (s *Service) GetChat(requestingUserID int) (*[]types.Chat, error) {
	return s.repo.GetChat(requestingUserID)
}
func (s *Service) GetSuggestedUsers(requestingUserID int) (*[]types.User, error) {
	return s.repo.GetSuggestedUsers(requestingUserID)
}
func (s *Service) SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error) {
	return s.repo.SearchChatByName(requestingUserID, chatName)
}
func (s *Service) AddChatUser(requestingUserID int, chatID int, userID int) (int, error) {
	return s.repo.AddChatUser(requestingUserID, chatID, userID)
}
func (s *Service) DeleteChatUser(requestingUserID int, chatID int, userID int) error {
	return s.repo.DeleteChatUser(requestingUserID, chatID, userID)
}
func (s *Service) GetChatUser(requestingUserID int, chatID int) (*[]types.User, error) {
	return s.repo.GetChatUser(requestingUserID, chatID)
}
func (s *Service) DeleteChatUserByUserID(userID int) error {
	return s.repo.DeleteChatUserByUserID(userID)
}
func (s *Service) GetChatUserByUserID(requestingUserID int, chatID int, userID int) (int, error) {
	return s.repo.GetChatUserByUserID(requestingUserID, chatID, userID)
}

func (s *Service) SearchForNewUsers(requestingUserID int, chatID int, search_term string) (*[]types.User, error) {
	if len(search_term) > 24 {
		return &[]types.User{}, nil
	}
	return s.repo.SearchForNewUsers(requestingUserID, chatID, search_term)
}

func (s *Service) GetChatIDsForChannels() ([]int, error) {
	return s.repo.GetChatIDsForChannels()
}

// Message
func (s *Service) CreateMessage(requestingUserID int, chatID int, text string) (int, error) {
	if len(text) < 1 {
		return -1, errors.New("Message must be at least 1 character.")
	}
	if len(text) > 1024 {
		return -1, errors.New("Message must be less than 1024 characters.")
	}
	return s.repo.CreateMessage(requestingUserID, chatID, text)
}
func (s *Service) GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.MessageDetail, error) {
	return s.repo.GetMessagesByChatID(requestingUserID, chatID)
}
func (s *Service) GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.MessageDetail, error) {
	return s.repo.GetMessageByID(requestingUserID, chatID, messageID)
}
func (s *Service) GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error) {
	return s.repo.GetChatPreview(requestingUserID)
}
func (s *Service) DeleteMessageByUserID(userID int) error {
	return s.repo.DeleteMessageByUserID(userID)
}
