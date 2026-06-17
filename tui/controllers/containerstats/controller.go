package containerstats

import (
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/docker/docker/api/types/container"
	"github.com/kernaxis/gmd/cachalot"
)

type StatsMsg struct {
	ID    string
	Stats container.StatsResponse
}

type Controller struct {
	mu         sync.RWMutex
	cli        *cachalot.Client
	pool       pond.Pool
	delay      time.Duration
	containers map[string]struct{}
	events     chan StatsMsg
}

func New() *Controller {
	pool := pond.NewPool(5, pond.WithQueueSize(5))
	c := &Controller{
		pool:       pool,
		delay:      500 * time.Millisecond,
		containers: make(map[string]struct{}),
		events:     make(chan StatsMsg),
	}
	return c
}

// SetClient wires the cachalot client once it has finished connecting.
// Until this is called, poll() has nothing to query against, which is fine
// since Start() is not currently invoked anywhere.
func (c *Controller) SetClient(cli *cachalot.Client) {
	c.mu.Lock()
	c.cli = cli
	c.mu.Unlock()
}

func (c *Controller) Start() {
	go c.loop()
}

func (c *Controller) Events() <-chan StatsMsg {
	return c.events
}

func (c *Controller) AddContainer(id string) {
	c.mu.Lock()
	c.containers[id] = struct{}{}
	c.mu.Unlock()
}

func (c *Controller) RemoveContainer(id string) {
	c.mu.Lock()
	delete(c.containers, id)
	c.mu.Unlock()
}

func (c *Controller) loop() {
	ticker := time.NewTicker(c.delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.poll()
		}
	}
}

func (c *Controller) poll() {
	c.mu.RLock()
	cli := c.cli
	ids := make([]string, 0, len(c.containers))
	for id := range c.containers {
		ids = append(ids, id)
	}
	c.mu.RUnlock()

	for _, id := range ids {
		cid := id
		c.pool.Submit(func() {
			stats, err := cli.ContainerStats(cid)
			if err != nil {
				return
			}
			c.events <- StatsMsg{ID: cid, Stats: stats}
		})
	}
}
