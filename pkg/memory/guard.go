// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package memory monitors process memory and cancels registered searches under pressure.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	log                          = logging.Log()
	meter                        = otel.Meter("korrel8r/memory")
	metricPressureTransitions, _ = meter.Int64Counter("memory_pressure.transitions", metric.WithDescription("Number of transitions into memory pressure"))
	metricSearchCancellations, _ = meter.Int64Counter("memory_pressure.search_cancellations", metric.WithDescription("Number of searches canceled due to memory pressure"))
	metricActiveSearches, _      = meter.Int64UpDownCounter("memory_pressure.active_searches", metric.WithDescription("Number of searches registered with the memory pressure guard"))
)

// Sampler reports current and maximum memory in bytes.
type Sampler interface {
	Sample() (used, limit uint64, err error)
}

// Guard tracks active searches and cancels them while memory pressure is high.
type Guard struct {
	sampler  Sampler
	high     int
	reset    int
	limit    uint64
	interval time.Duration

	mu           sync.Mutex
	pressured    bool
	sampleFailed bool
	nextID       uint64
	active       map[uint64]func()
}

// New creates a Guard. A zero high watermark disables it.
func New(sampler Sampler, high, reset int, limit uint64, interval time.Duration) (*Guard, error) {
	if high == 0 {
		return &Guard{}, nil
	}
	if sampler == nil {
		return nil, fmt.Errorf("memory pressure sampler is required")
	}
	if high < 1 || high > 100 {
		return nil, fmt.Errorf("memory pressure limit must be between 1 and 100")
	}
	if reset < 1 || reset >= high {
		return nil, fmt.Errorf("memory pressure reset must be between 1 and limit-1")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("memory pressure check interval must be positive")
	}
	return &Guard{sampler: sampler, high: high, reset: reset, limit: limit, interval: interval, active: map[uint64]func(){}}, nil
}

// NewFromTuning creates a system-memory guard from engine tuning.
// Call [Guard.Run] for the application lifetime and pass the guard to engine.Builder.SearchGuard.
func NewFromTuning(tuning *config.Tuning) (*Guard, error) {
	if tuning == nil {
		tuning = &config.Tuning{}
	}
	if tuning.MemoryLimit < 0 {
		return New(nil, 0, 0, 0, 0)
	}
	return New(
		NewSystemSampler(tuning.MemoryLimit > 0),
		tuning.GetMemoryPressureLimit(),
		tuning.GetMemoryPressureReset(),
		uint64(tuning.MemoryLimit),
		200*time.Millisecond,
	)
}

// Register adds a search. Searches registered during pressure are canceled immediately.
func (g *Guard) Register(cancel func()) func() {
	if g.sampler == nil {
		return func() {}
	}
	var cancelOnce sync.Once
	cancelSearch := func() {
		cancelOnce.Do(func() {
			metricSearchCancellations.Add(context.Background(), 1)
			cancel()
		})
	}
	g.mu.Lock()
	g.nextID++
	id := g.nextID
	g.active[id] = cancelSearch
	pressured := g.pressured
	g.mu.Unlock()
	metricActiveSearches.Add(context.Background(), 1)
	if pressured {
		cancelSearch()
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			delete(g.active, id)
			g.mu.Unlock()
			metricActiveSearches.Add(context.Background(), -1)
		})
	}
}

// Run samples until ctx is canceled.
func (g *Guard) Run(ctx context.Context) {
	if g.sampler == nil {
		return
	}
	g.sample()
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.sample()
		}
	}
}

func (g *Guard) sample() {
	used, limit, err := g.sampler.Sample()
	if err != nil {
		g.mu.Lock()
		firstFailure := !g.sampleFailed
		g.sampleFailed = true
		g.mu.Unlock()
		if firstFailure {
			log.V(1).Info("Cannot sample cgroup memory; memory pressure guard is inactive", "error", err)
		}
		return
	}
	g.mu.Lock()
	g.sampleFailed = false
	g.mu.Unlock()
	if limit == 0 {
		return
	}
	percent := int(float64(min(used, limit)) * 100 / float64(limit))
	high := percent >= g.high
	reset := percent <= g.reset
	if g.limit > 0 {
		high = used >= g.limit
		reset = used <= g.limit*uint64(g.reset)/uint64(g.high)
	}
	var cancels []func()
	g.mu.Lock()
	switch {
	case !g.pressured && high:
		g.pressured = true
		log.V(1).Info("Memory pressure is canceling active searches", "used", used, "limit", limit, "percent", percent)
		metricPressureTransitions.Add(context.Background(), 1)
		cancels = make([]func(), 0, len(g.active))
		for _, cancel := range g.active {
			cancels = append(cancels, cancel)
		}
	case g.pressured && reset:
		g.pressured = false
		log.V(2).Info("Memory pressure cleared", "used", used, "limit", limit, "percent", percent)
	}
	g.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}
