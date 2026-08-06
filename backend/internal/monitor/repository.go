// Package monitor enforces ownership and persistence boundaries for URL checks.
package monitor

import (
	"context"
	"errors"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidInput = errors.New("invalid monitor input")

// Store isolates monitor use cases from the database implementation.
type Store interface {
	Create(context.Context, models.Monitor) error
	List(context.Context, uuid.UUID) ([]models.Monitor, error)
}

// GormStore adapts GORM persistence to Store.
type GormStore struct{ db *gorm.DB }

// NewGormStore wires monitor persistence to a GORM database.
func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

// Create persists a monitor with its already authenticated owner.
func (store *GormStore) Create(ctx context.Context, value models.Monitor) error {
	return store.db.WithContext(ctx).Create(&value).Error
}

// List scopes results by owner and preserves creation order for stable API responses.
func (store *GormStore) List(ctx context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	var monitors []models.Monitor
	err := store.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC").Find(&monitors).Error
	return monitors, err
}
