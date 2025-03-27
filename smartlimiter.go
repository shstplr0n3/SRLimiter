package srlimiter

import (
	"sort"
	"sync"
)

type Load struct {
	priorityWeight uint16
	process        interface{}
}

type Collector struct {
	mutex sync.Mutex
	loads []*Load
}

func NewLoad(process interface{}, priority uint16) *Load {
	return &Load{
		process:        process,
		priorityWeight: priority,
	}
}

func (c *Collector) GetNextLoad() *Load {
	if len(c.loads) == 0 {
		return nil
	}
	load := c.loads[0]
	c.loads = c.loads[1:]
	return load
}

func (c *Collector) sortByPriority() *Collector {
	if len(c.loads) < 2 {
		return c
	}
	sort.Slice(c.loads, func(i, j int) bool {
		return c.loads[i].priorityWeight > c.loads[j].priorityWeight
	})
	return c
}

// func (c *Collector) Distribute
