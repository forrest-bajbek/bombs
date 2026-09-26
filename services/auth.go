package services

import "errors"

func (s *Service) CheckPassword(username string, password string) (int, error) {
	if len(username) < 2 || len(username) > 24 || len(password) < 5 || len(password) > 64 {
		return -1, errors.New("username or password is incorrect")
	}
	return s.repo.CheckPassword(username, password)
}

func (s *Service) ChangePassword(username string, old_password string, new_password string) error {
	if len(new_password) < 5 || len(new_password) > 64 {
		return errors.New("New password must be between 5 and 64 characters.")
	}
	return s.repo.ChangePassword(username, old_password, new_password)
}

func (s *Service) EnsureAdmin() error {
	return s.repo.EnsureAdmin()
}
