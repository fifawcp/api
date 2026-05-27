package domain

import (
	"context"
	"time"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID        string    `json:"id" example:"3f01faf6-d1a5-494a-bd41-b0475b6c8d03"`
	FirstName string    `json:"first_name" example:"John"`
	LastName  string    `json:"last_name" example:"Doe"`
	Username  string    `json:"username" example:"johndoe"`
	Email     string    `json:"email" example:"john.doe@example.com"`
	Role      UserRole  `json:"role" example:"user"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type UserUpdate struct {
	FirstName *string
	LastName  *string
	Username  *string
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByIdentifier(ctx context.Context, identifier string) (*User, error)
	GetUserByID(ctx context.Context, userID string) (*User, error)
	UpdateUser(ctx context.Context, userID string, updates UserUpdate) (*User, error)
}

type UserStorage interface {
	GetUser(ctx context.Context, userID string) (*User, error)
	SetUser(ctx context.Context, user *User) error
}
