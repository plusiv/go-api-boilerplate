// Package gormstore implements GORM persistence adapters.
package gormstore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
	"gorm.io/gorm"
)

type txHandle struct {
	db     *gorm.DB
	owner  *gorm.DB
	active atomic.Bool
}

// Transactioner runs work using GORM transactions.
type Transactioner struct{ db *gorm.DB }

// NewTransactioner creates a transaction runner for db.
func NewTransactioner(db *gorm.DB) *Transactioner { return &Transactioner{db: db} }

// Transaction commits if fn returns nil and rolls back on error or panic.
func (t *Transactioner) Transaction(ctx context.Context, fn func(tx any) error) error {
	if fn == nil {
		return errors.New("transaction callback is required")
	}
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		handle := &txHandle{db: tx, owner: t.db}
		handle.active.Store(true)
		defer handle.active.Store(false)
		return fn(handle)
	}); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	return nil
}

func transactionDB(tx any, owner *gorm.DB) (*gorm.DB, error) {
	handle, ok := tx.(*txHandle)
	if !ok || handle == nil || handle.db == nil || handle.owner != owner || !handle.active.Load() {
		return nil, errors.New("invalid transaction handle")
	}
	return handle.db, nil
}

var _ repositories.Transactioner = (*Transactioner)(nil)
