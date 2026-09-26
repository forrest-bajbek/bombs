package services

import "github.com/forrest-bajbek/bombs/types"

func (s *Service) GetFile(requestingUserID int, chatID int, fileID int) (*types.File, error) {
	return s.repo.GetFile(requestingUserID, chatID, fileID)
}

func (s *Service) StorageUsedBytes() (int64, error) {
	return s.repo.StorageUsedBytes()
}
