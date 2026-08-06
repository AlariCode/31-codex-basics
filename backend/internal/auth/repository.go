// Package auth keeps authentication rules independent from persistence and HTTP transport.
package auth

import (
	"context"
	"errors"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrEmailTaken lets the HTTP layer report a registration conflict without exposing database errors.
	ErrEmailTaken = errors.New("email is already registered")
	// ErrInvalidCredentials intentionally covers lookup and password failures uniformly.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidRefreshToken covers every unusable refresh-token state.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// UserStore isolates authentication rules from the database implementation.
type UserStore interface {
	Create(context.Context, models.User) error
	FindByEmail(context.Context, string) (models.User, error)
	FindByID(context.Context, uuid.UUID) (models.User, error)
	UpdateName(context.Context, uuid.UUID, string) (models.User, error)
	UpdateAvatar(context.Context, uuid.UUID, string) (models.User, error)
}

// SessionStore provides the atomic operations required to prevent refresh-token reuse.
type SessionStore interface {
	Create(context.Context, models.RefreshSession) error
	Rotate(context.Context, string, models.RefreshSession, time.Time) (models.User, error)
	Revoke(context.Context, string, time.Time) error
}

// GormUserStore adapts GORM persistence to UserStore.
type GormUserStore struct{ db *gorm.DB }

// NewGormUserStore wires the production user store to a GORM database.
func NewGormUserStore(db *gorm.DB) *GormUserStore { return &GormUserStore{db: db} }

// Create persists a user and translates a unique-email violation to ErrEmailTaken.
func (store *GormUserStore) Create(ctx context.Context, user models.User) error {
	err := store.db.WithContext(ctx).Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	return err
}

// FindByEmail returns ErrInvalidCredentials for an unknown email so login errors stay indistinguishable.
func (store *GormUserStore) FindByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := store.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrInvalidCredentials
	}
	return user, err
}

// FindByID returns ErrInvalidCredentials when the identity no longer exists.
func (store *GormUserStore) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user models.User
	err := store.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrInvalidCredentials
	}
	return user, err
}

// UpdateName updates the name and reloads the row so callers receive the canonical user record.
func (store *GormUserStore) UpdateName(ctx context.Context, id uuid.UUID, name string) (models.User, error) {
	result := store.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("name", name)
	if result.Error != nil {
		return models.User{}, result.Error
	}
	if result.RowsAffected != 1 {
		return models.User{}, ErrInvalidCredentials
	}
	return store.FindByID(ctx, id)
}

// UpdateAvatar updates the filename and reloads the row for the same response contract as UpdateName.
func (store *GormUserStore) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarPath string) (models.User, error) {
	result := store.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("avatar_path", avatarPath)
	if result.Error != nil {
		return models.User{}, result.Error
	}
	if result.RowsAffected != 1 {
		return models.User{}, ErrInvalidCredentials
	}
	return store.FindByID(ctx, id)
}

// GormSessionStore adapts GORM persistence to SessionStore.
type GormSessionStore struct{ db *gorm.DB }

// NewGormSessionStore wires refresh-session persistence to a GORM database.
func NewGormSessionStore(db *gorm.DB) *GormSessionStore { return &GormSessionStore{db: db} }

// Create persists the hashed token so the raw refresh token is never stored.
func (store *GormSessionStore) Create(ctx context.Context, session models.RefreshSession) error {
	return store.db.WithContext(ctx).Create(&session).Error
}

// Rotate revokes oldHash and creates its replacement in one transaction, preventing token reuse races.
func (store *GormSessionStore) Rotate(ctx context.Context, oldHash string, replacement models.RefreshSession, now time.Time) (models.User, error) {
	var user models.User
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session models.RefreshSession
		err := tx.Preload("User").Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", oldHash, now).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidRefreshToken
		}
		if err != nil {
			return err
		}
		result := tx.Model(&models.RefreshSession{}).Where("id = ? AND revoked_at IS NULL", session.ID).Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidRefreshToken
		}
		replacement.UserID = session.UserID
		if err := tx.Create(&replacement).Error; err != nil {
			return err
		}
		user = session.User
		return nil
	})
	return user, err
}

// Revoke is idempotent so logout succeeds even when the session is already revoked.
func (store *GormSessionStore) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	return store.db.WithContext(ctx).Model(&models.RefreshSession{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now).Error
}

// NewSession creates a session with a fresh identifier for token rotation and persistence.
func NewSession(userID uuid.UUID, tokenHash string, expiresAt time.Time) models.RefreshSession {
	return models.RefreshSession{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
}
