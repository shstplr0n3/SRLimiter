package srlimiter

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Rule defines validation logic for requests
type Rule func(*http.Request) bool

// Collector manages request collection and validation
type Collector struct {
	rules []Rule
	queue chan *http.Request
	mu    sync.RWMutex
}

// Distributor handles rate limiting and request processing
type Distributor struct {
	collector      *Collector
	perSecond      int
	perMinute      int
	workerPool     chan struct{}
	stopChan       chan struct{}
	wg             sync.WaitGroup
	minuteTicker   *time.Ticker
	minuteCount    atomic.Int32
	activeRequests atomic.Int32
}

// Middleware connects components for HTTP integration
type Middleware struct {
	collector   *Collector
	distributor *Distributor
}

// NewCollector creates a request collector
func NewCollector(rules ...Rule) *Collector {
	return &Collector{
		rules: rules,
		queue: make(chan *http.Request, 1000),
	}
}

// AddRule adds new validation rule
func (c *Collector) AddRule(r Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = append(c.rules, r)
}

// AddRequest validates and queues the request
func (c *Collector) AddRequest(req *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.rules {
		if !rule(req) {
			return
		}
	}

	select {
	case c.queue <- req:
	default:
		// Handle queue overflow
	}
}

// NewDistributor creates rate limiting processor
func NewDistributor(c *Collector, perSec, perMin int) (*Distributor, error) {
	if perSec < 1 {
		return nil, errors.New("perSecond must be at least 1")
	}

	d := &Distributor{
		collector:    c,
		perSecond:    perSec,
		perMinute:    perMin,
		workerPool:   make(chan struct{}, perSec),
		stopChan:     make(chan struct{}),
		minuteTicker: time.NewTicker(time.Minute),
	}

	d.wg.Add(1)
	go d.dispatch()
	return d, nil
}

// Middleware initialization
func NewMiddleware(rules []Rule, perSec, perMin int) *Middleware {
	collector := NewCollector(rules...)
	distributor, _ := NewDistributor(collector, perSec, perMin)
	return &Middleware{
		collector:   collector,
		distributor: distributor,
	}
}

// HTTP middleware handler
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.collector.AddRequest(r)

		select {
		case <-m.distributor.workerPool:
			defer func() { m.distributor.workerPool <- struct{}{} }()
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		}
	})
}

// Distributor logic
func (d *Distributor) dispatch() {
	defer d.wg.Done()
	defer d.minuteTicker.Stop()

	ticker := time.NewTicker(time.Second / time.Duration(d.perSecond))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.tryProcessRequest()
		case <-d.minuteTicker.C:
			d.minuteCount.Store(0)
		case <-d.stopChan:
			return
		}
	}
}

func (d *Distributor) tryProcessRequest() {
	if d.perMinute > 0 && d.minuteCount.Load() >= int32(d.perMinute) {
		return
	}

	select {
	case req := <-d.collector.queue:
		d.minuteCount.Add(1)
		d.activeRequests.Add(1)
		go d.process(req)
	default:
	}
}

func (d *Distributor) process(req *http.Request) {
	defer d.activeRequests.Add(-1)
	// Actual request processing logic
}

func (d *Distributor) Stop() {
	close(d.stopChan)
	d.wg.Wait()
}
