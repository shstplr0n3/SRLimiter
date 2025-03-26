package srlimiter

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Types of package
type Rule func(*http.Request) bool

type Collector struct {
	rules []Rule
	queue chan *http.Request
	mu    sync.RWMutex
}

type Distributor struct {
	collector    *Collector
	perSecond    int
	perMinute    int
	workerPool   chan struct{}
	stopChan     chan struct{}
	wg           sync.WaitGroup
	minuteTicker *time.Ticker
	minuteCount  int
	minuteMu     sync.Mutex
	handler      http.Handler
}

type Middleware struct {
	collector   *Collector
	distributor *Distributor
}

// Enhanced constructor for Middleware
func NewMiddleware(handler http.Handler, perSecond, perMinute int, rules ...Rule) (*Middleware, error) {
	collector := NewCollector(rules...)
	distributor, err := NewDistributor(collector, perSecond, perMinute)
	if err != nil {
		return nil, err
	}
	distributor.handler = handler

	return &Middleware{
		collector:   collector,
		distributor: distributor,
	}, nil
}

// ServeHTTP implementation for Middleware
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.collector.AddRequest(r)
}

// Enhanced process method with actual request handling
func (d *Distributor) process(req *http.Request) {
	defer func() { <-d.workerPool }()

	if d.handler != nil {
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()

		req = req.WithContext(ctx)
		d.handler.ServeHTTP(nil, req)
	}
}

// Enhanced Collector with request validation
func (c *Collector) ValidateRequest(req *http.Request) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.rules {
		if !rule(req) {
			return errors.New("request validation failed")
		}
	}
	return nil
}

// Helper function to create common rules
func IPRateLimit(limit int) Rule {
	visits := make(map[string]int)
	var mu sync.Mutex

	return func(r *http.Request) bool {
		ip := r.RemoteAddr
		mu.Lock()
		defer mu.Unlock()

		if visits[ip] >= limit {
			return false
		}
		visits[ip]++
		return true
	}
}

// Metrics collection
type Metrics struct {
	TotalRequests     uint64
	RejectedRequests  uint64
	ProcessedRequests uint64
}

func (d *Distributor) GetMetrics() *Metrics {
	return &Metrics{
		TotalRequests:     atomic.LoadUint64(&d.metrics.TotalRequests),
		RejectedRequests:  atomic.LoadUint64(&d.metrics.RejectedRequests),
		ProcessedRequests: atomic.LoadUint64(&d.metrics.ProcessedRequests),
	}
}

func (d *Distributor) GracefulShutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		d.Stop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
