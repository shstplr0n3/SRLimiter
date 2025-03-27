package pkg

import (
	"sync"
	"sync/atomic"
)

type Load struct {
	priorityWeight uint16
	process        interface{}
}

// type Load constructor
func NewLoad(weight uint16, process interface{}) *Load {
	return &Load{
		priorityWeight: weight,
		process:        process,
	}
}

// type Collector based on binary heap
type Collector struct {
	mutex  sync.RWMutex
	loads  []*Load
	length uint32
	pool   sync.Pool
}

// type Collector constructor
func NewCollector() *Collector {
	return &Collector{
		pool: sync.Pool{
			New: func() interface{} { return new(Load) },
		},
	}
}

// heap's direction up control
func (c *Collector) up(index int) {
	for {
		parent := (index - 1) / 2
		if parent == index ||
			c.loads[index].priorityWeight >= c.loads[parent].priorityWeight {
			break
		}
		c.loads[parent], c.loads[index] = c.loads[index], c.loads[parent]
		index = parent
	}
}

// heap's direction down control
func (c *Collector) down(index int) {
	last := len(c.loads) - 1
	for {
		left := 2*index + 1
		if left > last {
			break
		}
		smallest := left

		if right := left + 1; right <= last &&
			c.loads[right].priorityWeight < c.loads[left].priorityWeight {
			smallest = right
		}

		if c.loads[index].priorityWeight <= c.loads[smallest].priorityWeight {
			break
		}

		c.loads[index], c.loads[smallest] = c.loads[smallest], c.loads[index]
		index = smallest
	}
}

// addition of new element to object of type Collector
func (c *Collector) Add(elem interface{}) {
	load, ok := elem.(*Load)
	if !ok {
		panic("invalid type: expected *Load")
	}

	newLoad := c.pool.Get().(*Load)
	*newLoad = *load

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.loads = append(c.loads, newLoad)
	atomic.StoreUint32(&c.length, uint32(len(c.loads)))
	c.up(len(c.loads) - 1)
}

// Popping the last element of object of type Collector
func (c *Collector) Pop() *Load {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.loads) == 0 {
		return nil
	}

	root := c.loads[0]
	last := len(c.loads) - 1
	c.loads[0] = c.loads[last]
	c.loads = c.loads[:last]
	atomic.StoreUint32(&c.length, uint32(last))

	if last > 0 {
		c.down(0)
	}

	c.pool.Put(root)
	return root
}
