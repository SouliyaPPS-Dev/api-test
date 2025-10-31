package auth

import (
	"context"
	"time"
)

// UserRepository defines persistence operations for auth users.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, filter UserFilter) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error
}

// UserFilter allows narrowing user queries.
type UserFilter struct {
	Role UserRole
}
