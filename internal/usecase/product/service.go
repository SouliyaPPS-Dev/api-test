package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "backoffice/backend/internal/domain/product"

	"github.com/google/uuid"
)

// Service encapsulates product use cases.
type Service struct {
	repo    domain.Repository
	nowFunc func() time.Time
}

// NewService constructs a product service.
func NewService(repo domain.Repository) *Service {
	return &Service{
		repo:    repo,
		nowFunc: time.Now,
	}
}

// CreateInput contains the payload required for product creation.
type CreateInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SKU         string  `json:"sku"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

// UpdateInput encapsulates partial product updates.
type UpdateInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	SKU         *string  `json:"sku"`
	Price       *float64 `json:"price"`
	Quantity    *int     `json:"quantity"`
}

// Create stores a new product after validation.
func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.Product, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.SKU = strings.TrimSpace(input.SKU)
	if input.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.SKU == "" {
		return nil, errors.New("sku is required")
	}

	if _, err := s.repo.GetBySKU(ctx, input.SKU); err == nil {
		return nil, domain.ErrDuplicateSKU
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := s.nowFunc().UTC()
	product := &domain.Product{
		ID:          uuid.NewString(),
		Name:        input.Name,
		Description: input.Description,
		SKU:         input.SKU,
		Price:       input.Price,
		Quantity:    input.Quantity,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

// List retrieves all products.
func (s *Service) List(ctx context.Context) ([]*domain.Product, error) {
	return s.repo.List(ctx)
}

// Get fetches a product by id.
func (s *Service) Get(ctx context.Context, id string) (*domain.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.GetByID(ctx, id)
}

// Update applies partial updates to a product.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*domain.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.SKU != nil {
		newSKU := strings.TrimSpace(*input.SKU)
		if newSKU == "" {
			return nil, errors.New("sku cannot be empty")
		}
		if newSKU != product.SKU {
			if _, err := s.repo.GetBySKU(ctx, newSKU); err == nil {
				return nil, domain.ErrDuplicateSKU
			} else if !errors.Is(err, domain.ErrNotFound) {
				return nil, err
			}
		}
		*input.SKU = newSKU
	}

	product.Update(input.Name, input.Description, input.SKU, input.Price, input.Quantity)

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

// Delete removes a product.
func (s *Service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.Delete(ctx, id)
}
