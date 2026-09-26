package services

import (
	"errors"

	"github.com/forrest-bajbek/bombs/types"
)

func (s *Service) CreateChat(requestingUserID int, chatName string) (int, error) {
	if len(chatName) < 1 || len(chatName) > 32 {
		return -1, errors.New("Chat name must be between 1 and 32 characters.")
	}
	return s.repo.CreateChat(requestingUserID, chatName)
}

func (s *Service) IsUserInChat(userID int, chatID int) (bool, error) {
	return s.repo.IsUserInChat(userID, chatID)
}

func (s *Service) GetChatIDsForChannels() ([]int, error) {
	return s.repo.GetChatIDsForChannels()
}

func (s *Service) GetChatPreview(requestingUserID int) (*[]types.ChatPreview, error) {
	return s.repo.GetChatPreview(requestingUserID)
}

func (s *Service) UpdateChat(requestingUserID int, chatID int, chatName string) (*types.Chat, error) {
	if len(chatName) < 1 || len(chatName) > 32 {
		return nil, errors.New("Chat name must be between 1 and 32 characters.")
	}
	return s.repo.UpdateChat(requestingUserID, chatID, chatName)
}

func (s *Service) GetChatByID(requestingUserID int, chatID int) (*types.Chat, error) {
	return s.repo.GetChatByID(requestingUserID, chatID)
}
func (s *Service) GetChat(requestingUserID int) (*[]types.Chat, error) {
	return s.repo.GetChat(requestingUserID)
}

func (s *Service) SearchChatByName(requestingUserID int, chatName string) (*[]types.Chat, error) {
	return s.repo.SearchChatByName(requestingUserID, chatName)
}

func (s *Service) BombChat(requestingUserID int, chatID int) error {
	return s.repo.BombChat(requestingUserID, chatID)
}

func (s *Service) DeleteChat(requestingUserID int, chatID int) error {
	return s.repo.DeleteChat(requestingUserID, chatID)
}
