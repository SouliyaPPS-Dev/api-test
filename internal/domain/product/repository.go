package product

import "context"

// Repository defines persistence behaviours for products.
type Repository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	GetBySKU(ctx context.Context, sku string) (*Product, error)
	List(ctx context.Context) ([]*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
}
