package srlimiter

import "sort"

type Load struct {
	priorityWeight uint16
	process        interface{}
}

type Collector struct {
	loads []*Load
}

func NewLoad(process interface{}, priority uint16) *Load {
	return &Load{
		process:        process,
		priorityWeight: priority,
	}
}

func (l *Load) GetProcess() interface{} {
	return l.process
}

func (l *Load) GetPriority() uint16 {
	return l.priorityWeight
}

func NewCollector() *Collector {
	return &Collector{
		loads: make([]*Load, 0),
	}
}

func (c *Collector) AddLoad(load *Load) {
	c.loads = append(c.loads, load)
	c.sortByPriority()
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
