package main

import (
	"time"
)

type clickGuard struct {
	deadline time.Time
	moved    bool
}

func (c *clickGuard) Move() {
	c.moved = true
}

func (c *clickGuard) DragStart() {
	c.moved = false
}

func (c *clickGuard) DragEnd() {
	c.deadline = time.Now().Add(100 * time.Millisecond)
}

func (c *clickGuard) Click() bool {
	return c.deadline.IsZero() || !c.moved || c.deadline.Before(time.Now())
}
