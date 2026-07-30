// Package monitor manages user-owned monitoring point configurations.
package monitor

import (
	"context"
	"errors"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidInput = errors.New("invalid monitor input")

// Store persists monitoring points.
type Store interface {
	Create(context.Context, models.Monitor) error
	List(context.Context, uuid.UUID) ([]models.Monitor, error)
}

// GormStore is a PostgreSQL-backed monitor store.
type GormStore struct{ db *gorm.DB }

// NewGormStore creates a monitor store.
func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

// Create persists a monitor.
func (store *GormStore) Create(ctx context.Context, value models.Monitor) error {
	return store.db.WithContext(ctx).Create(&value).Error
}

// List returns a user's monitors ordered by creation time.
func (store *GormStore) List(ctx context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	var monitors []models.Monitor
	err := store.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC").Find(&monitors).Error
	return monitors, err
}
