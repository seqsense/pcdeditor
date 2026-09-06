//go:build !js
// +build !js

package main

import (
	"testing"
)

func TestHistoryUndoKeepsStepOnError(t *testing.T) {
	h := newHistory(4)
	h.steps = []undoStep{{memRecord{0xFF}}} // broken record data
	if _, ok := h.undo(nil); ok {
		t.Fatal("undo of a broken step must fail")
	}
	if len(h.steps) != 1 {
		t.Fatal("a broken step must not be dropped; a later undo would apply an older record to a mismatched state")
	}
}
