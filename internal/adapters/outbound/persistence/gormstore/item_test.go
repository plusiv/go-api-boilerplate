package gormstore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&Item{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestTransactions(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	runner := NewTransactioner(db)
	repo := NewItems(db)
	second := NewItems(db)
	var captured any
	if err := runner.Transaction(ctx, func(tx any) error {
		captured = tx
		a, err := repo.WithTx(tx)
		if err != nil {
			return err
		}
		b, err := second.WithTx(tx)
		if err != nil {
			return err
		}
		if err := a.Create(ctx, entities.Item{ID: "one", Name: "first"}); err != nil {
			return err
		}
		item, err := b.Get(ctx, "one")
		if err != nil {
			return fmt.Errorf("transaction read: %w", err)
		}
		if item.Name != "first" {
			return fmt.Errorf("unexpected item: %v", item)
		}
		return b.Create(ctx, entities.Item{ID: "two", Name: "second"})
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, "two"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.WithTx(captured); err == nil {
		t.Fatal("expired transaction accepted")
	}
	if _, err := repo.WithTx(nil); err == nil {
		t.Fatal("nil transaction accepted")
	}
	if _, err := repo.WithTx(db); err == nil {
		t.Fatal("non-transaction handle accepted")
	}
	rollback := errors.New("rollback")
	err := runner.Transaction(ctx, func(tx any) error {
		a, err := repo.WithTx(tx)
		if err != nil {
			return err
		}
		if err := a.Create(ctx, entities.Item{ID: "discard", Name: "discard"}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error: %v", err)
	}
	_, err = repo.Get(ctx, "discard")
	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperrors.NotFound {
		t.Fatalf("rollback did not discard item: %v", err)
	}
	func() {
		defer func() {
			if recovered := recover(); recovered != "transaction panic" {
				t.Errorf("unexpected panic: %v", recovered)
			}
		}()
		_ = runner.Transaction(ctx, func(tx any) error {
			bound, bindErr := repo.WithTx(tx)
			if bindErr != nil {
				t.Fatal(bindErr)
			}
			if createErr := bound.Create(ctx, entities.Item{ID: "panic", Name: "discard"}); createErr != nil {
				t.Fatal(createErr)
			}
			panic("transaction panic")
		})
	}()
	if _, err := repo.Get(ctx, "panic"); !errors.As(err, &appErr) || appErr.Kind != apperrors.NotFound {
		t.Fatalf("panic rollback: %v", err)
	}
	other := NewItems(testDB(t))
	if err := runner.Transaction(ctx, func(tx any) error {
		if _, err := other.WithTx(tx); err == nil {
			return errors.New("wrong database accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := runner.Transaction(canceled, func(any) error { return errors.New("callback should not run") }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestItemsFilters(t *testing.T) {
	ctx := context.Background()
	repo := NewItems(testDB(t))
	for _, item := range []entities.Item{{ID: "c", Name: "red"}, {ID: "a", Name: "red"}, {ID: "b", Name: "blue"}} {
		if err := repo.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(ctx, entities.Item{ID: "a"}); err == nil {
		t.Fatal("duplicate accepted")
	} else {
		var appErr *apperrors.Error
		if !errors.As(err, &appErr) || appErr.Kind != apperrors.Conflict {
			t.Fatalf("duplicate: %v", err)
		}
	}
	red, blue := "red", "blue"
	for _, tt := range []struct {
		name    string
		filters repositories.ItemFilters
		page    repositories.Page
		want    []entities.Item
	}{
		{"page", repositories.ItemFilters{}, repositories.Page{Limit: 1, Offset: 1}, []entities.Item{{ID: "b", Name: "blue"}}},
		{"combined", repositories.ItemFilters{Name: &repositories.Filter[string]{In: []string{red, blue}, IsNot: &blue}}, repositories.Page{Limit: 10}, []entities.Item{{ID: "a", Name: red}, {ID: "c", Name: red}}},
		{"empty in", repositories.ItemFilters{Name: &repositories.Filter[string]{In: []string{}}}, repositories.Page{Limit: 10}, []entities.Item{}},
		{"exclude", repositories.ItemFilters{Name: &repositories.Filter[string]{NotIn: []string{red}}}, repositories.Page{Limit: 10}, []entities.Item{{ID: "b", Name: blue}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.List(ctx, tt.filters, tt.page)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
