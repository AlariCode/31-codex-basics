package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

func TestStatsWeightedPercentAndEmptyBuckets(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	window, _ := periodWindow("1h", now)
	id := uuid.New()
	counts := []minuteCount{
		{MonitorID: id, BucketStart: window.From, Successes: 1, Failures: 1},
		{MonitorID: id, BucketStart: window.From.Add(time.Minute), Successes: 8},
	}
	result := buildStats([]models.Monitor{{ID: id}}, counts, window)
	stats := result.Monitors[0]
	if *stats.UptimePercent != 90 || len(stats.Points) != 60 {
		t.Fatalf("stats=%+v", stats)
	}
	if *stats.Points[0].UptimePercent != 50 || stats.Points[2].UptimePercent != nil {
		t.Fatal("incorrect empty/partial bucket")
	}
	for _, period := range []string{"1h", "24h", "7d", "30d"} {
		window, ok := periodWindow(period, now.Add(30*time.Second))
		if !ok || !window.From.Equal(window.From.Truncate(time.Minute)) {
			t.Fatalf("period %s", period)
		}
		result := buildStats([]models.Monitor{{ID: id}}, []minuteCount{}, window)
		points := result.Monitors[0].Points
		if len(points) > 169 || !points[0].Start.Equal(window.From) || !points[len(points)-1].End.Equal(window.To) {
			t.Fatalf("bounds for %s", period)
		}
	}
}

type statsStoreFake struct{ userID uuid.UUID }

func (s *statsStoreFake) List(_ context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	s.userID = userID
	return []models.Monitor{}, nil
}
func (s *statsStoreFake) readCounts(_ context.Context, userID uuid.UUID, _ statsWindow) ([]minuteCount, error) {
	if userID != s.userID {
		panic("owner scope lost")
	}
	return []minuteCount{}, nil
}

func TestStatsEndpointValidatesPeriodAndOwner(t *testing.T) {
	userID := uuid.New()
	store := &statsStoreFake{}
	controller := NewController(fakeAuthenticator{userID: userID}, &fakeStore{})
	controller.statistics = store
	mux := http.NewServeMux()
	controller.RegisterRoutes(mux)
	for _, test := range []struct {
		query string
		code  int
	}{
		{query: "24h", code: 200}, {query: "30d", code: 200}, {query: "1y", code: 400},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/monitors/stats?period="+test.query, nil))
		if w.Code != test.code {
			t.Fatalf("code=%d body=%s", w.Code, w.Body)
		}
	}
	if store.userID != userID {
		t.Fatal("wrong owner")
	}
}
