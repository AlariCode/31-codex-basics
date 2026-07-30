package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"uptime-backend/internal/database"
	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

func TestGormStores_RefreshRotation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	db, err := database.Open(databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	defer sqlDB.Close()
	if err := db.Exec("TRUNCATE refresh_sessions, users").Error; err != nil {
		t.Fatalf("truncate auth tables (apply migrations first): %v", err)
	}

	ctx := context.Background()
	users := NewGormUserStore(db)
	sessions := NewGormSessionStore(db)
	user := models.User{ID: uuid.New(), Email: "person@example.com", Name: "Person", PasswordHash: "hash"}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := users.Create(ctx, user); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected duplicate user error, got %v", err)
	}
	now := time.Now().UTC()
	if err := sessions.Create(ctx, NewSession(user.ID, "old-hash", now.Add(time.Hour))); err != nil {
		t.Fatalf("create session: %v", err)
	}
	rotatedUser, err := sessions.Rotate(ctx, "old-hash", NewSession(uuid.Nil, "new-hash", now.Add(time.Hour)), now)
	if err != nil || rotatedUser.ID != user.ID {
		t.Fatalf("rotate session: user=%#v err=%v", rotatedUser, err)
	}
	if _, err := sessions.Rotate(ctx, "old-hash", NewSession(uuid.Nil, "second-hash", now.Add(time.Hour)), now); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}
}
