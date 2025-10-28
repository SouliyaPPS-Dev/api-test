package postgres

import (
	"context"
	"errors"

	domain "backoffice/backend/internal/domain/product"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRepository persists products in PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository constructs a repository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

// Create inserts a new product.
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	const query = `
INSERT INTO products (id, name, description, sku, price, quantity, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	_, err := r.pool.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.SKU,
		product.Price,
		product.Quantity,
		product.CreatedAt,
		product.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateSKU
		}
		return err
	}
	return nil
}

// GetByID fetches a product by id.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	const query = `
SELECT id, name, description, sku, price, quantity, created_at, updated_at
FROM products WHERE id = $1
`
	row := r.pool.QueryRow(ctx, query, id)
	product, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

// GetBySKU fetches a product using its SKU.
func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	const query = `
SELECT id, name, description, sku, price, quantity, created_at, updated_at
FROM products WHERE sku = $1
`
	row := r.pool.QueryRow(ctx, query, sku)
	product, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

// List returns all products sorted by name.
func (r *ProductRepository) List(ctx context.Context) ([]*domain.Product, error) {
	const query = `
SELECT id, name, description, sku, price, quantity, created_at, updated_at
FROM products
ORDER BY name ASC
`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

// Update writes product updates to the database.
func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	const query = `
UPDATE products
SET name = $2,
    description = $3,
    sku = $4,
    price = $5,
    quantity = $6,
    updated_at = $7
WHERE id = $1
`
	tag, err := r.pool.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.SKU,
		product.Price,
		product.Quantity,
		product.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateSKU
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a product by id.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM products WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanProduct(row pgx.Row) (*domain.Product, error) {
	var p domain.Product
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.SKU,
		&p.Price,
		&p.Quantity,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
