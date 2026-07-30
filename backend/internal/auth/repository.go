// Package auth implements user authentication and session rotation.
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
	// ErrEmailTaken means an account already uses this email address.
	ErrEmailTaken = errors.New("email is already registered")
	// ErrInvalidCredentials means supplied login credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidRefreshToken means a refresh token is unknown, expired, or revoked.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// UserStore persists and retrieves users.
type UserStore interface {
	Create(context.Context, models.User) error
	FindByEmail(context.Context, string) (models.User, error)
	FindByID(context.Context, uuid.UUID) (models.User, error)
	UpdateName(context.Context, uuid.UUID, string) (models.User, error)
}

// SessionStore persists refresh sessions and rotates them atomically.
type SessionStore interface {
	Create(context.Context, models.RefreshSession) error
	Rotate(context.Context, string, models.RefreshSession, time.Time) (models.User, error)
	Revoke(context.Context, string, time.Time) error
}

// GormUserStore is a PostgreSQL-backed user store.
type GormUserStore struct{ db *gorm.DB }

// NewGormUserStore creates a GORM user store.
func NewGormUserStore(db *gorm.DB) *GormUserStore { return &GormUserStore{db: db} }

// Create persists a new user.
func (store *GormUserStore) Create(ctx context.Context, user models.User) error {
	err := store.db.WithContext(ctx).Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	return err
}

// FindByEmail finds a user by normalized email.
func (store *GormUserStore) FindByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := store.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrInvalidCredentials
	}
	return user, err
}

// FindByID finds a user by identifier.
func (store *GormUserStore) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user models.User
	err := store.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrInvalidCredentials
	}
	return user, err
}

// UpdateName changes a user's display name and returns the updated user.
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

// GormSessionStore is a PostgreSQL-backed refresh-session store.
type GormSessionStore struct{ db *gorm.DB }

// NewGormSessionStore creates a GORM refresh-session store.
func NewGormSessionStore(db *gorm.DB) *GormSessionStore { return &GormSessionStore{db: db} }

// Create persists a new refresh session.
func (store *GormSessionStore) Create(ctx context.Context, session models.RefreshSession) error {
	return store.db.WithContext(ctx).Create(&session).Error
}

// Rotate revokes the token matching oldHash and persists replacement atomically.
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

// Revoke invalidates an active refresh session when it exists.
func (store *GormSessionStore) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	return store.db.WithContext(ctx).Model(&models.RefreshSession{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now).Error
}

// NewSession creates a persistable refresh-session model.
func NewSession(userID uuid.UUID, tokenHash string, expiresAt time.Time) models.RefreshSession {
	return models.RefreshSession{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
}
