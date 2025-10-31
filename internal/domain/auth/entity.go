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
	// ErrInvalidRole indicates the provided role is not supported.
	ErrInvalidRole = errors.New("invalid role")
	// ErrPasswordMismatch indicates the current password is incorrect.
	ErrPasswordMismatch = errors.New("current password does not match")
	// ErrPasswordUnchanged indicates the new password matches the current one.
	ErrPasswordUnchanged = errors.New("new password must be different from current password")
)

// UserRole identifies the privileges assigned to a user.
type UserRole string

const (
	// RoleUser represents a standard application user.
	RoleUser UserRole = "user"
	// RoleAdmin represents an administrative user.
	RoleAdmin UserRole = "admin"
)

// User models the authentication entity persisted in storage.
type User struct {
	ID           string
	Email        string
	Name         string
	Role         UserRole
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Credentials captures raw credential input for login.
type Credentials struct {
	Email    string
	Password string
}
