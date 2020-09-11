package main

import (
	"testing"
	"time"
)

func TestClickGuard(t *testing.T) {
	cg := &clickGuard{}

	if !cg.Click() {
		t.Error("First click event must be treated as click")
	}
	if !cg.Click() {
		t.Error("Second click event must be treated as click")
	}

	cg.DragStart()
	if !cg.Click() {
		t.Error("Click during drag must be treated as click")
	}
	cg.DragEnd()
	if !cg.Click() {
		t.Error("Click right after dragging without move must be treated as click")
	}

	cg.DragStart()
	if !cg.Click() {
		t.Error("Click during drag must be treated as click")
	}
	cg.Move()
	if cg.Click() {
		t.Error("Click during drag must not be treated as click")
	}
	cg.DragEnd()
	if cg.Click() {
		t.Error("Click right after dragging must not be treated as click")
	}

	time.Sleep(clickGuardDuration + time.Millisecond)
	if !cg.Click() {
		t.Error("Click after guard duration must be treated as click")
	}
}
