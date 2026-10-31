// Package repositories defines persistence ports and query types.
package repositories

import (
	"context"

	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
)

// Filter combines all present predicates with AND. Nil fields are omitted.
// An empty In slice matches nothing; nil means no membership constraint.
type Filter[T comparable] struct {
	Is    *T
	IsNot *T
	In    []T
	NotIn []T
}

// ItemFilters contains supported item predicates.
type ItemFilters struct {
	ID   *Filter[string]
	Name *Filter[string]
}

// Page requests a bounded offset page. Offset is zero-based.
type Page struct{ Limit, Offset int }

// ItemRepository is the persistence port for the example use case.
// Create returns a conflict error for duplicate IDs; Get returns a not-found error for missing IDs.
// List returns items sorted by ID, with filters applied before pagination.
type ItemRepository interface {
	Create(context.Context, entities.Item) error
	Get(context.Context, string) (entities.Item, error)
	List(context.Context, ItemFilters, Page) ([]entities.Item, error)
}
