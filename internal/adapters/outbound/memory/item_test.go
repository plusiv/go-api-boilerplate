package memory

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
)

func TestItems(t *testing.T) {
	ctx := context.Background()
	repo := NewItems()
	for _, item := range []entities.Item{{ID: "c", Name: "red"}, {ID: "a", Name: "red"}, {ID: "b", Name: "blue"}} {
		if err := repo.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	var appErr *apperrors.Error
	if err := repo.Create(ctx, entities.Item{ID: "a"}); !errors.As(err, &appErr) || appErr.Kind != apperrors.Conflict {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := repo.Get(ctx, "missing"); !errors.As(err, &appErr) || appErr.Kind != apperrors.NotFound {
		t.Fatalf("missing: %v", err)
	}
	red, blue := "red", "blue"
	tests := []struct {
		name    string
		filters repositories.ItemFilters
		page    repositories.Page
		want    []entities.Item
	}{
		{"sorted page", repositories.ItemFilters{}, repositories.Page{Limit: 1, Offset: 1}, []entities.Item{{ID: "b", Name: "blue"}}},
		{"combined predicates", repositories.ItemFilters{Name: &repositories.Filter[string]{In: []string{red, blue}, IsNot: &blue}}, repositories.Page{Limit: 10}, []entities.Item{{ID: "a", Name: "red"}, {ID: "c", Name: "red"}}},
		{"empty in", repositories.ItemFilters{Name: &repositories.Filter[string]{In: []string{}}}, repositories.Page{Limit: 10}, []entities.Item{}},
		{"not in", repositories.ItemFilters{Name: &repositories.Filter[string]{NotIn: []string{red}}}, repositories.Page{Limit: 10}, []entities.Item{{ID: "b", Name: "blue"}}},
		{"offset beyond end", repositories.ItemFilters{}, repositories.Page{Limit: 10, Offset: 100}, []entities.Item{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.List(ctx, tt.filters, tt.page)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.List(canceled, repositories.ItemFilters{}, repositories.Page{Limit: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled list: %v", err)
	}
}
