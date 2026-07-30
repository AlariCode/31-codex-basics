// Package database configures the PostgreSQL connection.
package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open connects GORM to PostgreSQL.
func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
}
