// Package usecases implements application operations independently of transport.
package usecases

import (
	"context"
	"errors"
	"strings"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
	"github.com/rotisserie/eris"
)

// Items provides example item operations.
type Items struct{ repo repositories.ItemRepository }

// NewItems creates item use cases using the provided repository.
func NewItems(repo repositories.ItemRepository) *Items { return &Items{repo: repo} }

// Create validates and stores an item.
func (s *Items) Create(ctx context.Context, id, name string) (entities.Item, error) {
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if id == "" || name == "" || len(id) > 100 || len(name) > 200 {
		return entities.Item{}, apperrors.NewValidation("invalid input")
	}
	item := entities.Item{ID: id, Name: name}
	if err := s.repo.Create(ctx, item); err != nil {
		return entities.Item{}, repositoryError(err, "create item")
	}
	return item, nil
}

// Get retrieves an item by ID.
func (s *Items) Get(ctx context.Context, id string) (entities.Item, error) {
	if strings.TrimSpace(id) == "" {
		return entities.Item{}, apperrors.NewValidation("invalid input")
	}
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return entities.Item{}, repositoryError(err, "get item")
	}
	return item, nil
}

// List retrieves a bounded page of matching items.
func (s *Items) List(ctx context.Context, filters repositories.ItemFilters, page repositories.Page) ([]entities.Item, error) {
	if page.Limit < 1 || page.Limit > 100 || page.Offset < 0 {
		return nil, apperrors.NewValidation("invalid input")
	}
	items, err := s.repo.List(ctx, filters, page)
	if err != nil {
		return nil, repositoryError(err, "list items")
	}
	return items, nil
}

// Preserve expected failures and cancellation; unexpected dependencies get a private cause.
func repositoryError(err error, operation string) error {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return apperrors.NewInternal(eris.Wrap(err, operation))
}
