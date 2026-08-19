package audit

import (
	"fmt"
	"sync"
)

// Checkpoint tracks how far each subscriber has read the stream.
type Checkpoint struct {
	mu      sync.RWMutex
	offsets map[string]int
}

func NewCheckpoint() *Checkpoint { return &Checkpoint{offsets: map[string]int{}} }

func (c *Checkpoint) Get(subID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offsets[subID]
}

func (c *Checkpoint) Set(subID string, offset int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.offsets[subID] = offset
}

// Advance moves the checkpoint forward only when the new offset is greater.
func (c *Checkpoint) Advance(subID string, offset int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offset < c.offsets[subID] {
		return fmt.Errorf("offset %d regressed below %d", offset, c.offsets[subID])
	}
	c.offsets[subID] = offset
	return nil
}
