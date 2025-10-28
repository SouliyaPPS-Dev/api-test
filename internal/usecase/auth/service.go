package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "backoffice/backend/internal/domain/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service coordinates authentication workflows between domain and infrastructure.
type Service struct {
	users   domain.UserRepository
	tokens  TokenManager
	nowFunc func() time.Time
}

// NewService constructs an auth service.
func NewService(users domain.UserRepository, tokens TokenManager) *Service {
	return &Service{
		users:   users,
		tokens:  tokens,
		nowFunc: time.Now,
	}
}

// Register creates a new user and returns the persisted entity without a password hash.
func (s *Service) Register(ctx context.Context, email, password, name string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailExists
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := s.nowFunc().UTC()
	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		Name:         strings.TrimSpace(name),
		PasswordHash: string(hashed),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return sanitizeUser(user), nil
}

// Login validates credentials and returns a token plus user.
func (s *Service) Login(ctx context.Context, creds domain.Credentials) (string, *domain.User, error) {
	email := strings.TrimSpace(strings.ToLower(creds.Email))
	password := strings.TrimSpace(creds.Password)
	if email == "" || password == "" {
		return "", nil, domain.ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", nil, domain.ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, sanitizeUser(user), nil
}

// VerifyToken validates a bearer token and returns the associated user.
func (s *Service) VerifyToken(ctx context.Context, token string) (*domain.User, error) {
	userID, err := s.tokens.Validate(token)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, err
	}

	return sanitizeUser(user), nil
}

func sanitizeUser(u *domain.User) *domain.User {
	if u == nil {
		return nil
	}
	copy := *u
	copy.PasswordHash = ""
	return &copy
}
