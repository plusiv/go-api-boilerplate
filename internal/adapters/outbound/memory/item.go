// Package memory provides ephemeral, concurrency-safe repository adapters.
package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
)

// Items stores items in memory. Data is lost when the process stops.
type Items struct {
	mu    sync.RWMutex
	items map[string]entities.Item
}

// NewItems creates an empty repository.
func NewItems() *Items { return &Items{items: make(map[string]entities.Item)} }

// Create inserts an item unless its ID exists.
func (r *Items) Create(ctx context.Context, item entities.Item) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create item: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create item: %w", err)
	}
	if _, ok := r.items[item.ID]; ok {
		return apperrors.NewConflict("item already exists")
	}
	r.items[item.ID] = item
	return nil
}

// Get finds an item by ID.
func (r *Items) Get(ctx context.Context, id string) (entities.Item, error) {
	if err := ctx.Err(); err != nil {
		return entities.Item{}, fmt.Errorf("get item: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return entities.Item{}, fmt.Errorf("get item: %w", err)
	}
	item, ok := r.items[id]
	if !ok {
		return entities.Item{}, apperrors.NewNotFound("item")
	}
	return item, nil
}

// List filters, sorts, and then pages the matching items.
func (r *Items) List(ctx context.Context, filter repositories.ItemFilters, page repositories.Page) ([]entities.Item, error) {
	if page.Limit < 1 || page.Limit > 100 || page.Offset < 0 {
		return nil, apperrors.NewValidation("invalid page")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]entities.Item, 0)
	for _, item := range r.items {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("list items: %w", err)
		}
		if matches(item.ID, filter.ID) && matches(item.Name, filter.Name) {
			items = append(items, item)
		}
	}
	slices.SortFunc(items, func(a, b entities.Item) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if page.Offset >= len(items) {
		return []entities.Item{}, nil
	}
	items = items[page.Offset:]
	if len(items) > page.Limit {
		items = items[:page.Limit]
	}
	return items, nil
}

func matches[T comparable](value T, f *repositories.Filter[T]) bool {
	if f == nil {
		return true
	}
	return (f.Is == nil || value == *f.Is) && (f.IsNot == nil || value != *f.IsNot) &&
		(f.In == nil || slices.Contains(f.In, value)) && !slices.Contains(f.NotIn, value)
}

var _ repositories.ItemRepository = (*Items)(nil)
