package user

import (
	"context"
)

type Repository interface {
	List(ctx context.Context) ([]User, error)
	CreateUser(username string, password string) (int, error)
	GetUserByID(id int) (*User, error)
	GetUsers() (*[]User, error)
	GetUsersByUsername(username string) (*[]User, error)
	DeleteUserByID(id int) error
	CheckPassword(username string, password string) (int, error)
	ChangePassword(username string, old_password string, new_password string) error
	EnsureAdmin() error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// example
func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) CreateUser(username string, password string) (int, error) {
	return s.repo.CreateUser(username, password)
}
func (s *Service) GetUserByID(id int) (*User, error) {
	return s.repo.GetUserByID(id)
}
func (s *Service) GetUsers() (*[]User, error) {
	return s.repo.GetUsers()
}
func (s *Service) GetUsersByUsername(username string) (*[]User, error) {
	return s.repo.GetUsersByUsername(username)
}
func (s *Service) DeleteUserByID(id int) error {
	return s.repo.DeleteUserByID(id)
}
func (s *Service) CheckPassword(username string, password string) (int, error) {
	return s.repo.CheckPassword(username, password)
}
func (s *Service) ChangePassword(username string, old_password string, new_password string) error {
	return s.repo.ChangePassword(username, old_password, new_password)
}
func (s *Service) EnsureAdmin() error {
	return s.repo.EnsureAdmin()
}
