package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Database wraps the pgx connection pool.
type Database struct {
	Pool *pgxpool.Pool
}

// New establishes a new connection pool against the provided DSN.
func New(ctx context.Context, dsn string) (*Database, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Database{Pool: pool}, nil
}

// Close drains the connection pool.
func (db *Database) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
	}
}
