package domain

import (
	"context"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRepositoryInterface interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByIdentifier(ctx context.Context, identifier string) (*User, error)
}
