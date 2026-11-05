package gormstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
	"gorm.io/gorm"
)

// Item is the GORM persistence model for an example item.
type Item struct {
	ID   string `gorm:"primaryKey;not null"`
	Name string `gorm:"not null"`
}

// Items implements the item repository using GORM.
type Items struct {
	db    *gorm.DB
	owner *gorm.DB
}

// NewItems creates an item repository. Create its table with a migration before use.
func NewItems(db *gorm.DB) *Items { return &Items{db: db, owner: db} }

// WithTx returns a new repository that uses the supplied transaction.
func (r *Items) WithTx(tx any) (repositories.TransactionalItemRepository, error) {
	db, err := transactionDB(tx, r.owner)
	if err != nil {
		return nil, fmt.Errorf("bind item repository: %w", err)
	}
	return &Items{db: db, owner: r.owner}, nil
}

// Create inserts an item.
func (r *Items) Create(ctx context.Context, item entities.Item) error {
	row := Item{ID: item.ID, Name: item.Name}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperrors.NewConflict("item already exists")
		}
		return fmt.Errorf("create item: %w", err)
	}
	return nil
}

// Get finds an item by ID.
func (r *Items) Get(ctx context.Context, id string) (entities.Item, error) {
	var row Item
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Item{}, apperrors.NewNotFound("item")
		}
		return entities.Item{}, fmt.Errorf("get item: %w", err)
	}
	return entities.Item{ID: row.ID, Name: row.Name}, nil
}

// List applies all filters before sorting and paging.
func (r *Items) List(ctx context.Context, filters repositories.ItemFilters, page repositories.Page) ([]entities.Item, error) {
	if page.Limit < 1 || page.Limit > 100 || page.Offset < 0 {
		return nil, apperrors.NewValidation("invalid page")
	}
	query := r.db.WithContext(ctx).Model(&Item{})
	query = applyStringFilter(query, "id", filters.ID)
	query = applyStringFilter(query, "name", filters.Name)
	var rows []Item
	if err := query.Order("id ASC").Limit(page.Limit).Offset(page.Offset).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	items := make([]entities.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, entities.Item{ID: row.ID, Name: row.Name})
	}
	return items, nil
}

// Columns are constants chosen by the repository, never supplied by callers.
func applyStringFilter(db *gorm.DB, column string, filter *repositories.Filter[string]) *gorm.DB {
	if filter == nil {
		return db
	}
	if filter.Is != nil {
		db = db.Where(column+" = ?", *filter.Is)
	}
	if filter.IsNot != nil {
		db = db.Where(column+" <> ?", *filter.IsNot)
	}
	if filter.In != nil {
		if len(filter.In) == 0 {
			db = db.Where("1 = 0")
		} else {
			db = db.Where(column+" IN ?", filter.In)
		}
	}
	if len(filter.NotIn) > 0 {
		db = db.Where(column+" NOT IN ?", filter.NotIn)
	}
	return db
}

var _ repositories.TransactionalItemRepository = (*Items)(nil)
