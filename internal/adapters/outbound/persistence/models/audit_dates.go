// Package models contains persistence models used by database adapters.
package models

import (
	"time"

	"gorm.io/gorm"
)

// AuditDates adds creation, update, and soft-deletion timestamps to a GORM model.
// GORM excludes soft-deleted records from ordinary queries.
type AuditDates struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
