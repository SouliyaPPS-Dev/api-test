package auth

import (
	"errors"
	"time"
)

var (
	// ErrInvalidCredentials indicates a login failure.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrEmailExists signals a duplicate email registration.
	ErrEmailExists = errors.New("email already registered")
	// ErrTokenInvalid means a supplied token cannot be validated.
	ErrTokenInvalid = errors.New("token invalid or expired")
	// ErrUserNotFound indicates missing user.
	ErrUserNotFound = errors.New("user not found")
)

// User models the authentication entity persisted in storage.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Credentials captures raw credential input for login.
type Credentials struct {
	Email    string
	Password string
}
