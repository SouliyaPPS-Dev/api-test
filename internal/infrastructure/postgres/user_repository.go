package postgres

import (
	"context"
	"errors"
	"time"

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
INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Role,
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
SELECT id, email, name, role, password_hash, created_at, updated_at
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
SELECT id, email, name, role, password_hash, created_at, updated_at
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

// List returns users filtered by the provided criteria.
func (r *UserRepository) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, error) {
	query := `
SELECT id, email, name, role, password_hash, created_at, updated_at
FROM users
`
	var args []any
	if filter.Role != "" {
		query += "WHERE role = $1 "
		args = append(args, filter.Role)
	}
	query += "ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// Update modifies an existing user record.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
UPDATE users
SET email = $2, name = $3, role = $4, updated_at = $5
WHERE id = $1
`
	ct, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Role,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailExists
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Delete removes a user by id.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM users WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// UpdatePassword updates the stored password hash for a user.
func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	const query = `
UPDATE users
SET password_hash = $2, updated_at = $3
WHERE id = $1
`
	ct, err := r.pool.Exec(ctx, query, id, passwordHash, updatedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.Role,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
