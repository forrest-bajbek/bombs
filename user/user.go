package user

import "time"

type User struct {
	ID        int
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
	IsAdmin   bool
}
