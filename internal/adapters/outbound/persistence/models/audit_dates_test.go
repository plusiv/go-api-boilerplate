package models

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type auditTestRecord struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	AuditDates
}

func TestAuditDatesWithGORM(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{NowFunc: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&auditTestRecord{}); err != nil {
		t.Fatal(err)
	}
	record := auditTestRecord{Name: "first"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	if !record.CreatedAt.Equal(now) || !record.UpdatedAt.Equal(now) || record.DeletedAt.Valid {
		t.Fatalf("unexpected dates after create: %+v", record.AuditDates)
	}
	created := now
	now = now.Add(time.Hour)
	if err := db.Model(&record).Update("name", "second").Error; err != nil {
		t.Fatal(err)
	}
	if !record.CreatedAt.Equal(created) || !record.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected dates after update: %+v", record.AuditDates)
	}
	now = now.Add(time.Hour)
	if err := db.Delete(&record).Error; err != nil {
		t.Fatal(err)
	}
	var found auditTestRecord
	if err := db.First(&found, record.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted record should be hidden, got %v", err)
	}
	if err := db.Unscoped().First(&found, record.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !found.DeletedAt.Valid || !found.DeletedAt.Time.Equal(now) {
		t.Fatalf("unexpected deletion date: %+v", found.DeletedAt)
	}
	if err := db.Unscoped().Delete(&found).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().First(&found, record.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("record should be permanently deleted, got %v", err)
	}
}
