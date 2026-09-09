package monitor

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

type scheduleStore interface {
	ListAll(context.Context) ([]models.Monitor, error)
	Record(context.Context, models.Monitor, CheckResult) error
	Cleanup(context.Context, time.Time) error
}

type urlChecker interface {
	Check(context.Context, string) CheckResult
}

type scheduledCheck struct {
	value      models.Monitor
	next       time.Time
	cancel     context.CancelFunc
	deleted    bool
	restarting bool
}

// Scheduler owns schedules in one event loop; only checks and retention run concurrently.
// Deploy a single backend instance until distributed job ownership is implemented.
type Scheduler struct {
	store   scheduleStore
	checker urlChecker
	wake    chan struct{}
	entries map[uuid.UUID]*scheduledCheck
	done    chan *scheduledCheck
	active  int
	limit   int
	workers sync.WaitGroup
}

// NewScheduler creates a scheduler with twenty bounded workers and no queued duplicate checks.
func NewScheduler(store scheduleStore, checker *Checker) *Scheduler {
	return &Scheduler{
		store: store, checker: checker, wake: make(chan struct{}, 1),
		entries: map[uuid.UUID]*scheduledCheck{}, done: make(chan *scheduledCheck, 20), limit: 20,
	}
}

// Notify requests immediate reconciliation after a committed configuration mutation.
func (s *Scheduler) Notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Run restores persisted monitors and stops all background work before returning on cancellation.
func (s *Scheduler) Run(ctx context.Context) {
	tick := time.NewTicker(100 * time.Millisecond)
	reconcile := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	defer reconcile.Stop()
	s.workers.Add(1)
	go func() { defer s.workers.Done(); s.cleanup(ctx) }()
	defer func() {
		for _, entry := range s.entries {
			if entry.cancel != nil {
				entry.cancel()
			}
		}
		s.workers.Wait()
	}()
	s.reload(ctx, time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			s.reload(ctx, time.Now())
		case <-reconcile.C:
			s.reload(ctx, time.Now())
		case entry := <-s.done:
			s.active--
			entry.cancel()
			entry.cancel = nil
			entry.restarting = false
			if entry.deleted {
				delete(s.entries, entry.value.ID)
			}
		case <-tick.C:
		}
		if ctx.Err() != nil {
			return
		}
		s.dispatch(ctx, time.Now())
	}
}

func (s *Scheduler) reload(ctx context.Context, now time.Time) {
	readContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	values, err := s.store.ListAll(readContext)
	if err != nil {
		log.Print("monitor scheduler: configuration read failed")
		return
	}
	s.reconcile(values, now)
}

func (s *Scheduler) reconcile(values []models.Monitor, now time.Time) {
	seen := map[uuid.UUID]bool{}
	for _, value := range values {
		seen[value.ID] = true
		entry, exists := s.entries[value.ID]
		if !exists {
			s.entries[value.ID] = &scheduledCheck{value: value, next: now}
			continue
		}
		if entry.value.ConfigVersion == value.ConfigVersion {
			continue
		}
		if entry.cancel != nil {
			entry.cancel()
		}
		entry.restarting = true
		entry.value = value
		entry.next = now
	}
	for id, entry := range s.entries {
		if seen[id] {
			continue
		}
		entry.deleted = true
		if entry.cancel != nil {
			entry.cancel()
			continue
		}
		delete(s.entries, id)
	}
}

func (s *Scheduler) dispatch(ctx context.Context, now time.Time) {
	due := []*scheduledCheck{}
	for _, entry := range s.entries {
		if entry.deleted || entry.next.After(now) {
			continue
		}
		if entry.cancel != nil {
			if entry.restarting {
				continue
			}
			// A slow request skips elapsed starts instead of accumulating catch-up jobs.
			entry.next = now.Add(time.Duration(entry.value.IntervalSeconds) * time.Second)
			continue
		}
		due = append(due, entry)
	}
	sort.Slice(due, func(i, j int) bool { return due[i].next.Before(due[j].next) })
	for _, entry := range due {
		if s.active >= s.limit {
			break
		}
		checkContext, cancel := context.WithCancel(ctx)
		entry.cancel = cancel
		entry.next = now.Add(time.Duration(entry.value.IntervalSeconds) * time.Second)
		s.active++
		s.workers.Add(1)
		go s.execute(checkContext, entry, entry.value)
	}
}

func (s *Scheduler) execute(ctx context.Context, entry *scheduledCheck, value models.Monitor) {
	defer s.workers.Done()
	result := s.checker.Check(ctx, value.URL)
	if ctx.Err() == nil {
		writeContext, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := s.store.Record(writeContext, value, result); err != nil {
			log.Printf("monitor %s: result persistence failed", value.ID)
		}
		cancel()
	}
	// The channel holds every active worker so shutdown never depends on the event loop draining it.
	s.done <- entry
}

func (s *Scheduler) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if err := s.store.Cleanup(ctx, time.Now()); err != nil && ctx.Err() == nil {
			log.Print("monitor retention: cleanup failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
