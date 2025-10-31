package user

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "backoffice/backend/internal/domain/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service provides user management use cases for administrative workflows.
type Service struct {
	repo    domain.UserRepository
	nowFunc func() time.Time
}

// NewService constructs a user service around the provided repository.
func NewService(repo domain.UserRepository) *Service {
	return &Service{
		repo:    repo,
		nowFunc: time.Now,
	}
}

// Filter captures supported filters for listing users.
type Filter struct {
	Role string
}

// CreateInput defines the payload to create a new user.
type CreateInput struct {
	Email    string
	Name     string
	Password string
	Role     string
}

// UpdateInput defines the payload to update a user.
type UpdateInput struct {
	Email *string
	Name  *string
	Role  *string
}

// List returns users matching the supplied filter.
func (s *Service) List(ctx context.Context, filter Filter) ([]*domain.User, error) {
	domainFilter := domain.UserFilter{}
	if trimmed := strings.TrimSpace(strings.ToLower(filter.Role)); trimmed != "" {
		role, err := ensureRole(trimmed, false)
		if err != nil {
			return nil, err
		}
		domainFilter.Role = role
	}

	users, err := s.repo.List(ctx, domainFilter)
	if err != nil {
		return nil, err
	}
	return sanitizeUsers(users), nil
}

// Get retrieves a single user by its identifier.
func (s *Service) Get(ctx context.Context, id string) (*domain.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("user id is required")
	}
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return sanitizeUser(user), nil
}

// Create persists a new user with the provided details.
func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.Name)
	password := strings.TrimSpace(input.Password)
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	role, err := ensureRole(input.Role, true)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.GetByEmail(ctx, email); err == nil {
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
		Name:         name,
		Role:         role,
		PasswordHash: string(hashed),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return sanitizeUser(user), nil
}

// Update modifies the persisted user.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*domain.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("user id is required")
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*input.Email))
		if email == "" {
			return nil, errors.New("email is required")
		}
		user.Email = email
	}
	if input.Name != nil {
		user.Name = strings.TrimSpace(*input.Name)
	}
	if input.Role != nil {
		role, err := ensureRole(*input.Role, true)
		if err != nil {
			return nil, err
		}
		user.Role = role
	}

	user.UpdatedAt = s.nowFunc().UTC()
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return sanitizeUser(user), nil
}

// Delete removes the target user.
func (s *Service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("user id is required")
	}
	return s.repo.Delete(ctx, id)
}

func ensureRole(raw string, defaultToUser bool) (domain.UserRole, error) {
	role := domain.UserRole(strings.TrimSpace(strings.ToLower(raw)))
	if role == "" {
		if defaultToUser {
			return domain.RoleUser, nil
		}
		return "", nil
	}
	switch role {
	case domain.RoleUser, domain.RoleAdmin:
		return role, nil
	default:
		return "", domain.ErrInvalidRole
	}
}

func sanitizeUser(u *domain.User) *domain.User {
	if u == nil {
		return nil
	}
	copy := *u
	copy.PasswordHash = ""
	return &copy
}

func sanitizeUsers(items []*domain.User) []*domain.User {
	out := make([]*domain.User, 0, len(items))
	for _, item := range items {
		out = append(out, sanitizeUser(item))
	}
	return out
}
