package services

import "github.com/forrest-bajbek/bombs/types"

func (s *Service) AddChatUser(requestingUserID int, chatID int, userID int) (int, error) {
	return s.repo.AddChatUser(requestingUserID, chatID, userID)
}

func (s *Service) GetChatUser(requestingUserID int, chatID int) (*[]types.User, error) {
	return s.repo.GetChatUser(requestingUserID, chatID)
}

func (s *Service) GetChatUserByUserID(requestingUserID int, chatID int, userID int) (int, error) {
	return s.repo.GetChatUserByUserID(requestingUserID, chatID, userID)
}

func (s *Service) GetSuggestedUsers(requestingUserID int) (*[]types.User, error) {
	return s.repo.GetSuggestedUsers(requestingUserID)
}

func (s *Service) SearchForNewUsers(requestingUserID int, chatID int, search_term string) (*[]types.User, error) {
	if len(search_term) > 24 {
		return &[]types.User{}, nil
	}
	return s.repo.SearchForNewUsers(requestingUserID, chatID, search_term)
}

func (s *Service) DeleteChatUserByUserID(userID int) error {
	return s.repo.DeleteChatUserByUserID(userID)
}

func (s *Service) DeleteChatUser(requestingUserID int, chatID int, userID int) error {
	return s.repo.DeleteChatUser(requestingUserID, chatID, userID)
}
