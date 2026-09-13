package health

import (
	"context"
	"sync"
	"time"

	"myinternetvpn/client/internal/defaults"
)

// Checker probes VPN health and optionally reconnects.
type Checker struct {
	mu           sync.Mutex
	interval     time.Duration
	enabled      bool
	healthy      func() bool
	reconnect    func() error
	cancel       context.CancelFunc
	failures     int
	MaxFails     int
	reconnecting bool
	lastAttempt  time.Time
}

func New(interval time.Duration, healthy func() bool, reconnect func() error) *Checker {
	if interval <= 0 {
		interval = defaults.DefaultHealthInterval
	}
	return &Checker{
		interval:  interval,
		healthy:   healthy,
		reconnect: reconnect,
		MaxFails:  2,
	}
}

func (c *Checker) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.enabled = true
	go c.loop(ctx)
}

func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.enabled = false
	c.failures = 0
	c.reconnecting = false
}

func (c *Checker) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

func (c *Checker) SetInterval(d time.Duration) {
	if d <= 0 {
		d = defaults.DefaultHealthInterval
	}
	c.mu.Lock()
	c.interval = d
	c.mu.Unlock()
}

func (c *Checker) Failures() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures
}

func (c *Checker) loop(ctx context.Context) {
	for {
		c.mu.Lock()
		d := c.interval
		c.mu.Unlock()
		if d <= 0 {
			d = defaults.DefaultHealthInterval
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			c.tick()
		}
	}
}

func (c *Checker) tick() {
	c.mu.Lock()
	healthyFn := c.healthy
	reconnectFn := c.reconnect
	maxFails := c.MaxFails
	busy := c.reconnecting
	last := c.lastAttempt
	c.mu.Unlock()

	if healthyFn == nil || busy {
		return
	}
	if healthyFn() {
		c.mu.Lock()
		c.failures = 0
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.failures++
	fails := c.failures
	c.mu.Unlock()

	if fails < maxFails || reconnectFn == nil {
		return
	}
	if time.Since(last) < defaults.ReconnectBackoff {
		return
	}

	c.mu.Lock()
	c.reconnecting = true
	c.lastAttempt = time.Now()
	c.mu.Unlock()

	err := reconnectFn()

	c.mu.Lock()
	c.reconnecting = false
	if err == nil {
		c.failures = 0
	}
	// Keep failures on error so we don't reconnect-storm; backoff gates retries.
	c.mu.Unlock()
}

// TickOnce is used by tests.
func (c *Checker) TickOnce() {
	c.tick()
}
