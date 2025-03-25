package srlimiter

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// Types of package

// functor for request validation
type Rule func(*http.Request) bool

// Collector is a type for collecting requests in a queue with verification
type Collector struct {
	rules []Rule
	queue chan *http.Request
	mu    sync.RWMutex
}

// Distributor is a type for rate limiting and distributing incoming requests
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
}

type Middleware struct {
	collector   *Collector
	distributor *Distributor
}

// Methods for Collector

// Constructor of instances of type Collector
func NewCollector(rules ...Rule) *Collector {
	return &Collector{
		rules: rules,
		queue: make(chan *http.Request, 1000),
	}
}

// Method for addition of the new rule
func (c *Collector) AddRule(r Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = append(c.rules, r)
}

// Method for addition of the new request
func (c *Collector) AddRequest(req *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.rules {
		if !rule(req) {
			return
		}
	}
	c.queue <- req
}

// Methods for Distributor

// Constuctor of instances of type Distributor
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

// Method for dispatching requests from queue to gorutines
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
			d.resetMinuteCounter()
		case <-d.stopChan:
			return
		}
	}
}

func (d *Distributor) tryProcessRequest() {
	d.minuteMu.Lock()
	defer d.minuteMu.Unlock()

	if d.perMinute > 0 && d.minuteCount >= d.perMinute {
		return
	}

	select {
	case req := <-d.collector.queue:
		select {
		case d.workerPool <- struct{}{}:
			d.minuteCount++
			go d.process(req)
		default:
			go func() { d.collector.queue <- req }()
		}
	default:
	}
}

// Util funcs

func (d *Distributor) process(*http.Request) {
	return
}

func (d *Distributor) resetMinuteCounter() {
	d.minuteMu.Lock()
	defer d.minuteMu.Unlock()
	d.minuteCount = 0
}

func (d *Distributor) Stop() {
	close(d.stopChan)
	d.wg.Wait()
}
