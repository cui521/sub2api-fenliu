package api

import (
	"context"
	"sync"
	"time"
)

type probeBatchRunner func(ctx context.Context, limit int, model string, trigger string) ([]probeJob, error)

type probeControlState struct {
	Enabled         bool       `json:"enabled"`
	IntervalSeconds int        `json:"interval_seconds"`
	BatchSize       int        `json:"batch_size"`
	Model           string     `json:"model"`
	Running         bool       `json:"running"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastRunCount    int        `json:"last_run_count"`
	LastError       string     `json:"last_error,omitempty"`
}

type probeControlUpdate struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	IntervalSeconds int     `json:"interval_seconds,omitempty"`
	BatchSize       int     `json:"batch_size,omitempty"`
	Model           string  `json:"model,omitempty"`
}

type probeController struct {
	mu     sync.Mutex
	state  probeControlState
	cancel context.CancelFunc
	run    probeBatchRunner
}

func newProbeController(run probeBatchRunner) *probeController {
	return &probeController{
		state: probeControlState{
			Enabled:         false,
			IntervalSeconds: 60,
			BatchSize:       5,
			Model:           "gpt-4o-mini",
		},
		run: run,
	}
}

func (c *probeController) Snapshot() probeControlState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *probeController) Update(req probeControlUpdate) probeControlState {
	c.mu.Lock()
	if req.IntervalSeconds > 0 {
		c.state.IntervalSeconds = clamp(req.IntervalSeconds, 10, 3600)
	}
	if req.BatchSize > 0 {
		c.state.BatchSize = clamp(req.BatchSize, 1, 50)
	}
	if req.Model != "" {
		c.state.Model = req.Model
	}
	if req.Enabled != nil {
		c.state.Enabled = *req.Enabled
	}
	state := c.state
	wasRunning := c.state.Running
	c.mu.Unlock()

	if state.Enabled {
		if wasRunning {
			c.stop()
		}
		c.start()
	} else {
		c.stop()
	}
	return c.Snapshot()
}

func (c *probeController) RunOnce(ctx context.Context, trigger string) ([]probeJob, error) {
	state := c.Snapshot()
	jobs, err := c.run(ctx, state.BatchSize, state.Model, trigger)

	c.mu.Lock()
	now := time.Now().UTC()
	c.state.LastRunAt = &now
	c.state.LastRunCount = len(jobs)
	if err != nil {
		c.state.LastError = err.Error()
	} else {
		c.state.LastError = ""
	}
	c.mu.Unlock()

	return jobs, err
}

func (c *probeController) start() {
	c.mu.Lock()
	if c.state.Running {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.state.Running = true
	c.mu.Unlock()

	go c.loop(ctx)
}

func (c *probeController) stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.state.Running = false
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (c *probeController) loop(ctx context.Context) {
	_, _ = c.RunOnce(ctx, "auto_batch")
	for {
		state := c.Snapshot()
		timer := time.NewTimer(time.Duration(state.IntervalSeconds) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			_, _ = c.RunOnce(ctx, "auto_batch")
		}
	}
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
