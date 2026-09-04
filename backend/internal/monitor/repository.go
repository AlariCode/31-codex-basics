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
var ErrNotFound = errors.New("monitor not found")

// Store isolates monitor use cases from the database implementation.
type Store interface {
	Create(context.Context, models.Monitor) error
	List(context.Context, uuid.UUID) ([]models.Monitor, error)
	Update(context.Context, uuid.UUID, models.Monitor) error
	UpdateFavicon(context.Context, uuid.UUID, uuid.UUID, string) error
	Delete(context.Context, uuid.UUID, uuid.UUID) error
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

// Update changes only the editable fields of a monitor owned by the user.
func (store *GormStore) Update(ctx context.Context, userID uuid.UUID, value models.Monitor) error {
	result := store.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Where("id = ? AND user_id = ?", value.ID, userID).
		Updates(map[string]any{"url": value.URL, "interval_seconds": value.IntervalSeconds})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateFavicon stores the locally served favicon path for a monitor owned by the user.
func (store *GormStore) UpdateFavicon(ctx context.Context, userID, monitorID uuid.UUID, faviconPath string) error {
	result := store.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Where("id = ? AND user_id = ?", monitorID, userID).
		Update("favicon_path", faviconPath)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a monitor only when it belongs to the authenticated user.
func (store *GormStore) Delete(ctx context.Context, userID, monitorID uuid.UUID) error {
	result := store.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", monitorID, userID).
		Delete(&models.Monitor{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
