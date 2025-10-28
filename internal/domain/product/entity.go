package product

import (
	"errors"
	"time"
)

var (
	// ErrNotFound indicates a product could not be located.
	ErrNotFound = errors.New("product not found")
	// ErrDuplicateSKU signals SKU uniqueness constraint breaches.
	ErrDuplicateSKU = errors.New("product with SKU already exists")
)

// Product captures the state of an individual product.
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SKU         string    `json:"sku"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Update applies arbitrary field updates to the product.
func (p *Product) Update(name, description, sku *string, price *float64, quantity *int) {
	if name != nil {
		p.Name = *name
	}
	if description != nil {
		p.Description = *description
	}
	if sku != nil {
		p.SKU = *sku
	}
	if price != nil {
		p.Price = *price
	}
	if quantity != nil {
		p.Quantity = *quantity
	}
	p.UpdatedAt = time.Now().UTC()
}
