package health

import (
	"context"
	"sync"
	"time"
)

// Checker probes VPN health and optionally reconnects.
type Checker struct {
	mu       sync.Mutex
	interval time.Duration
	enabled  bool
	healthy  func() bool
	reconnect func() error
	cancel   context.CancelFunc
	failures int
	MaxFails int
}

func New(interval time.Duration, healthy func() bool, reconnect func() error) *Checker {
	if interval <= 0 {
		interval = 30 * time.Second
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
}

func (c *Checker) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

func (c *Checker) Failures() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures
}

func (c *Checker) loop(ctx context.Context) {
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
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
	c.mu.Unlock()

	if healthyFn == nil {
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
	_ = reconnectFn()
	c.mu.Lock()
	c.failures = 0
	c.mu.Unlock()
}

// TickOnce is used by tests.
func (c *Checker) TickOnce() {
	c.tick()
}
