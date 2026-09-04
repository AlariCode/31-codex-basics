// Package models contains the persistence shapes shared by services and repositories.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User stores the identity and password hash needed to authenticate an account.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"uniqueIndex;not null"`
	Name         string    `gorm:"not null"`
	PasswordHash string    `gorm:"not null"`
	AvatarPath   string    `gorm:"not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RefreshSession stores only a hash of the browser token and its revocation state.
type RefreshSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	User      User       `gorm:"constraint:OnDelete:CASCADE"`
	TokenHash string     `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Monitor binds a URL check to its owner so monitor data remains user-scoped.
type Monitor struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"`
	User            User      `gorm:"constraint:OnDelete:CASCADE"`
	URL             string    `gorm:"not null"`
	FaviconPath     string    `gorm:"not null;default:''"`
	IntervalSeconds int       `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
