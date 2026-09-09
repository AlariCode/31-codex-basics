package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

type blockingChecker struct{ started chan context.Context }

func (c blockingChecker) Check(ctx context.Context, _ string) CheckResult {
	c.started <- ctx
	<-ctx.Done()
	return CheckResult{}
}

type schedulerStoreFake struct {
	mu     sync.Mutex
	writes int
}

func (*schedulerStoreFake) ListAll(context.Context) ([]models.Monitor, error) {
	return []models.Monitor{}, nil
}
func (s *schedulerStoreFake) Record(context.Context, models.Monitor, CheckResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	return nil
}
func (*schedulerStoreFake) Cleanup(context.Context, time.Time) error { return nil }

func TestSchedulerRestartEditDeleteAndConcurrency(t *testing.T) {
	store := &schedulerStoreFake{}
	checker := blockingChecker{started: make(chan context.Context, 20)}
	s := NewScheduler(store, NewChecker())
	s.checker = checker
	s.limit = 2
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	first := models.Monitor{ID: uuid.New(), ConfigVersion: 1, URL: "https://example.com", IntervalSeconds: 5}
	second := models.Monitor{ID: uuid.New(), ConfigVersion: 1, URL: "https://example.org", IntervalSeconds: 5}
	third := models.Monitor{ID: uuid.New(), ConfigVersion: 1, URL: "https://example.net", IntervalSeconds: 5}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.reconcile([]models.Monitor{first, second}, now)
	s.dispatch(ctx, now)
	<-checker.started
	<-checker.started
	if s.active != 2 {
		t.Fatalf("active=%d", s.active)
	}
	s.reconcile([]models.Monitor{first, second, third}, now)
	s.dispatch(ctx, now.Add(time.Second))
	if s.active != 2 {
		t.Fatal("exceeded concurrency")
	}
	s.dispatch(ctx, now.Add(5*time.Second))
	if !s.entries[first.ID].next.Equal(now.Add(10 * time.Second)) {
		t.Fatal("slow check did not skip elapsed start")
	}
	// Editing an in-flight check with a week-long interval must still restart immediately.
	first.ConfigVersion++
	first.IntervalSeconds = 604800
	s.reconcile([]models.Monitor{first, second}, now.Add(6*time.Second))
	s.dispatch(ctx, now.Add(6*time.Second))
	finished := <-s.done
	if finished.value.ID != first.ID {
		t.Fatal("wrong check canceled")
	}
	finished.cancel()
	finished.cancel = nil
	finished.restarting = false
	s.active--
	s.dispatch(ctx, now.Add(6*time.Second))
	<-checker.started
	if !s.entries[first.ID].next.Equal(now.Add(6 * time.Second).Add(604800 * time.Second)) {
		t.Fatal("edit not dispatched immediately")
	}
	s.reconcile([]models.Monitor{}, now.Add(7*time.Second))
	for range 2 {
		finished := <-s.done
		if !finished.deleted {
			t.Fatal("deleted check remains")
		}
	}
	s.workers.Wait()
	if store.writes != 0 {
		t.Fatal("canceled checks persisted")
	}
}

type startupStore struct {
	schedulerStoreFake
	monitor models.Monitor
}

func (s *startupStore) ListAll(context.Context) ([]models.Monitor, error) {
	return []models.Monitor{s.monitor}, nil
}

func TestSchedulerRunRestoresAndShutsDown(t *testing.T) {
	store := &startupStore{monitor: models.Monitor{
		ID: uuid.New(), ConfigVersion: 1, URL: "https://example.com", IntervalSeconds: 3600,
	}}
	checker := blockingChecker{started: make(chan context.Context, 1)}
	scheduler := NewScheduler(store, NewChecker())
	scheduler.checker = checker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { scheduler.Run(ctx); close(done) }()
	select {
	case <-checker.started:
	case <-time.After(2 * time.Second):
		t.Fatal("startup did not restore existing monitor")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not wait for canceled checks")
	}
	if store.writes != 0 {
		t.Fatal("shutdown persisted a failure")
	}
}
