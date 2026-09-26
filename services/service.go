package services

import (
	"github.com/forrest-bajbek/bombs/types"
)

type Repo interface {
	// user
	CreateUser(username string, password string) (int, error)
	UserExists(username string) (bool, error)
	GetUserByID(userID int) (*types.User, error)
	GetUserByUsername(username string) (*types.User, error)
	GetUsers() (*[]types.User, error)
	SearchUsers(username string) (*[]types.User, error)
	DeleteUser(userID int) error

	// auth
	CheckPassword(username string, password string) (int, error)
	ChangePassword(username string, old_password string, new_password string) error
	EnsureAdmin() error

	// chat
	CreateChat(requestingUserID int, chatName string) (int, error)
	IsUserInChat(userID int, chatID int) (bool, error)
	GetChatIDsForChannels() ([]int, error)
	GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error)
	UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error)
	GetChatByID(requestingUserID int, chatID int) (*types.Chat, error)
	GetChat(requestingUserID int) (*[]types.Chat, error)
	SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error)
	BombChat(requestingUserID int, chatID int) error
	DeleteChat(requestingUserID int, chatID int) error

	// chatUser
	AddChatUser(requestingUserID int, chatID int, userID int) (int, error)
	GetChatUser(requestingUserID int, chatID int) (*[]types.User, error)
	GetChatUserByUserID(requestingUserID int, chatID int, userID int) (int, error)
	GetSuggestedUsers(requestingUserID int) (*[]types.User, error)
	SearchForNewUsers(requestingUserID int, chatID int, search_term string) (*[]types.User, error)
	DeleteChatUserByUserID(userID int) error
	DeleteChatUser(requestingUserID int, chatID int, userID int) error

	// message
	CreateMessage(requestingUserID int, chatID int, text string, files []types.NewFile) (int, error)
	GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.ChannelMessage, error)
	GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.ChannelMessage, error)
	DeleteMessageByUserID(userID int) error

	// file
	GetFile(requestingUserID int, chatID int, fileID int) (*types.File, error)
	StorageUsedBytes() (int64, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

