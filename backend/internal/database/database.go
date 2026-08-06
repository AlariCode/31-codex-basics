// Package database keeps database-driver setup in one place.
package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open enables GORM's translated errors so repositories can map database conflicts to domain errors.
func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
}
