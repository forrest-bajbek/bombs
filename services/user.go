package services

import (
	"errors"

	"github.com/forrest-bajbek/bombs/types"
)

func (s *Service) CreateUser(username string, password string) (int, error) {
	if len(username) < 2 || len(username) > 24 {
		return -1, errors.New("Username must be betwee 2 and 24 characters.")
	}
	if len(password) < 5 || len(password) > 64 {
		return -1, errors.New("Password must be between 5 and 64 characters.")
	}
	return s.repo.CreateUser(username, password)
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
