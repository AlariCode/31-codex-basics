package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"uptime-backend/internal/models"
)

// ListAll restores the scheduler from persisted configuration in stable order.
func (store *GormStore) ListAll(ctx context.Context) ([]models.Monitor, error) {
	values := []models.Monitor{}
	err := store.db.WithContext(ctx).Order("created_at, id").Find(&values).Error
	return values, err
}

// Record atomically updates the current observation and minute counters, fencing obsolete checks.
func (store *GormStore) Record(ctx context.Context, value models.Monitor, result CheckResult) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current models.Monitor
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", value.ID).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if current.ConfigVersion != value.ConfigVersion {
			return nil
		}
		err = tx.Model(&current).Updates(map[string]any{
			"last_checked_at": result.CheckedAt, "last_status": result.Status,
			"last_http_status": result.HTTPStatus, "last_error": result.Error,
		}).Error
		if err != nil || result.Status == "blocked" {
			return err
		}
		successes, failures := 0, 1
		if result.Status == "up" {
			successes, failures = 1, 0
		}
		return tx.Exec(`INSERT INTO monitor_minutes (monitor_id, bucket_start, successes, failures)
   VALUES (?, ?, ?, ?)
   ON CONFLICT (monitor_id, bucket_start) DO UPDATE
   SET successes = monitor_minutes.successes + EXCLUDED.successes,
       failures = monitor_minutes.failures + EXCLUDED.failures`,
			value.ID,
			result.CheckedAt.UTC().Truncate(time.Minute),
			successes,
			failures,
		).Error
	})
}

// Cleanup deletes expired buckets in bounded transactions; API reads enforce retention independently.
func (store *GormStore) Cleanup(ctx context.Context, now time.Time) error {
	cutoff := now.UTC().Add(-30 * 24 * time.Hour).Truncate(time.Minute)
	for {
		result := store.db.WithContext(ctx).Exec(`DELETE FROM monitor_minutes WHERE ctid IN
   (SELECT ctid FROM monitor_minutes WHERE bucket_start < ? LIMIT 5000)`, cutoff)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected < 5000 {
			return nil
		}
	}
}

type minuteCount struct {
	MonitorID   uuid.UUID
	BucketStart time.Time
	Successes   int64
	Failures    int64
}

func (store *GormStore) readCounts(ctx context.Context, userID uuid.UUID, window statsWindow) ([]minuteCount, error) {
	counts := []minuteCount{}
	// Boundaries are whole minutes because finer precision is deliberately not stored.
	err := store.db.WithContext(ctx).Raw(`SELECT m.monitor_id,
  date_bin(?::interval, m.bucket_start, '2000-01-01'::timestamptz) AS bucket_start,
  SUM(m.successes) AS successes, SUM(m.failures) AS failures
  FROM monitor_minutes m JOIN monitors ON monitors.id = m.monitor_id
  WHERE monitors.user_id = ? AND m.bucket_start >= ? AND m.bucket_start < ?
  GROUP BY m.monitor_id, 2 ORDER BY 2`,
		fmt.Sprintf("%d seconds", int(window.Step.Seconds())),
		userID,
		window.From,
		window.To,
	).Scan(&counts).Error
	return counts, err
}
