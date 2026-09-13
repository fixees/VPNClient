package subscription

import (
	"context"
	"sync"
	"time"
)

// Scheduler periodically refreshes remote subscriptions.
type Scheduler struct {
	mu       sync.Mutex
	interval time.Duration
	syncAll  func(ctx context.Context) error
	cancel   context.CancelFunc
}

func NewScheduler(interval time.Duration, syncAll func(ctx context.Context) error) *Scheduler {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &Scheduler{interval: interval, syncAll: syncAll}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop(ctx)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *Scheduler) SetInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d > 0 {
		s.interval = d
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	// Initial delayed sync so UI can finish startup.
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if s.syncAll != nil {
				_ = s.syncAll(ctx)
			}
			s.mu.Lock()
			interval := s.interval
			s.mu.Unlock()
			timer.Reset(interval)
		}
	}
}
