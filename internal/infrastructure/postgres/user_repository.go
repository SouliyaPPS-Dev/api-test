package postgres

import (
	"context"
	"errors"

	domain "backoffice/backend/internal/domain/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository persists users in PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository constructs a repository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailExists
		}
		return err
	}
	return nil
}

// GetByEmail fetches a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
SELECT id, email, name, password_hash, created_at, updated_at
FROM users WHERE email = $1
`
	row := r.pool.QueryRow(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
SELECT id, email, name, password_hash, created_at, updated_at
FROM users WHERE id = $1
`
	row := r.pool.QueryRow(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
