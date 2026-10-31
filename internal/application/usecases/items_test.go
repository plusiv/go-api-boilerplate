package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
)

type failedRepo struct{ err error }

func (r failedRepo) Create(context.Context, entities.Item) error { return r.err }
func (r failedRepo) Get(context.Context, string) (entities.Item, error) {
	return entities.Item{}, r.err
}
func (r failedRepo) List(context.Context, repositories.ItemFilters, repositories.Page) ([]entities.Item, error) {
	return nil, r.err
}

func TestRepositoryErrors(t *testing.T) {
	cause := errors.New("private storage error")
	for _, tt := range []struct {
		name  string
		cause error
		kind  apperrors.Kind
	}{
		{"unexpected", cause, apperrors.Internal},
		{"expected", apperrors.NewNotFound("item"), apperrors.NotFound},
		{"canceled", context.Canceled, ""},
		{"deadline", context.DeadlineExceeded, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewItems(failedRepo{err: tt.cause})
			_, err := svc.Get(context.Background(), "id")
			if !errors.Is(err, tt.cause) {
				t.Fatalf("cause not preserved: %v", err)
			}
			var appErr *apperrors.Error
			if tt.kind == "" {
				if errors.As(err, &appErr) {
					t.Fatalf("cancellation classified as application error: %v", err)
				}
				return
			}
			if !errors.As(err, &appErr) || appErr.Kind != tt.kind {
				t.Fatalf("unexpected classification: %v", err)
			}
		})
	}
}
