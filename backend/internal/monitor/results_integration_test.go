package monitor

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"uptime-backend/internal/database"
	"uptime-backend/internal/models"
)

func TestResultsPostgres(t *testing.T) {
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("set TEST_DATABASE_URL to an isolated migrated PostgreSQL database")
	}
	db, err := database.Open(connection)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	user := models.User{ID: uuid.New(), Email: uuid.NewString() + "@example.com", Name: "Monitor test", PasswordHash: "test"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&user)
	ctx := context.Background()
	store := NewGormStore(db)
	monitor := models.Monitor{ID: uuid.New(), UserID: user.ID, URL: "https://example.com", IntervalSeconds: 5, ConfigVersion: 1}
	if err := store.Create(ctx, monitor); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 12, 0, 30, 0, time.UTC)
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := "up"
			if i%3 == 0 {
				status = "down"
			}
			if err := store.Record(ctx, monitor, CheckResult{CheckedAt: now, Status: status}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	window, _ := periodWindow("24h", now.Add(time.Minute))
	counts, err := store.readCounts(ctx, user.ID, window)
	if err != nil || len(counts) != 1 || counts[0].Successes != 8 || counts[0].Failures != 4 {
		t.Fatalf("counts=%+v err=%v", counts, err)
	}
	other, err := store.readCounts(ctx, uuid.New(), window)
	if err != nil || len(other) != 0 {
		t.Fatal("history leaked to another owner")
	}
	monitor.URL = "https://example.org"
	if err := store.Update(ctx, user.ID, monitor); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(ctx, monitor, CheckResult{CheckedAt: now, Status: "down"}); err != nil {
		t.Fatal(err)
	}
	monitors, err := store.List(ctx, user.ID)
	if err != nil || monitors[0].LastStatus != "pending" || monitors[0].ConfigVersion != 2 {
		t.Fatalf("monitor=%+v err=%v", monitors, err)
	}
	monitor = monitors[0]
	if err := store.Record(ctx, monitor, CheckResult{CheckedAt: now, Status: "blocked", Error: "forbidden_address"}); err != nil {
		t.Fatal(err)
	}
	counts, _ = store.readCounts(ctx, user.ID, window)
	if counts[0].Successes != 8 || counts[0].Failures != 4 {
		t.Fatal("obsolete or blocked result changed history")
	}
	// A failed counter write must roll back the last observation as well.
	if err := db.Exec("UPDATE monitor_minutes SET successes = 2147483647 WHERE monitor_id = ?", monitor.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Record(ctx, monitor, CheckResult{CheckedAt: now, Status: "up"}); err == nil {
		t.Fatal("expected counter overflow")
	}
	monitors, _ = store.List(ctx, user.ID)
	if monitors[0].LastStatus != "blocked" {
		t.Fatal("last result committed without its aggregate")
	}
	if err := db.Exec("UPDATE monitor_minutes SET successes = 8 WHERE monitor_id = ?", monitor.ID).Error; err != nil {
		t.Fatal(err)
	}
	old := now.Add(-31 * 24 * time.Hour)
	if err := store.Record(ctx, monitor, CheckResult{CheckedAt: old, Status: "up"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Cleanup(ctx, now); err != nil {
		t.Fatal(err)
	}
	var total int64
	if err := db.Table("monitor_minutes").Where("monitor_id = ?", monitor.ID).Count(&total).Error; err != nil || total != 1 {
		t.Fatalf("total=%d err=%v", total, err)
	}
	if err := store.Delete(ctx, user.ID, monitor.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(ctx, monitor, CheckResult{CheckedAt: now, Status: "up"}); err != nil {
		t.Fatal(err)
	}
	db.Table("monitor_minutes").Where("monitor_id = ?", monitor.ID).Count(&total)
	if total != 0 {
		t.Fatal("deleted monitor retained history")
	}
}
