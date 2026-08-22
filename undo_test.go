//go:build !js
// +build !js

package main

import (
	"testing"
)

func TestHistoryUndoKeepsEntryOnError(t *testing.T) {
	h := &historyMem{
		maxHistory: 4,
		entries:    [][][]byte{{{0xFF}}}, // broken patch data
	}
	if _, ok := h.undo(nil); ok {
		t.Fatal("undo of a broken entry must fail")
	}
	if len(h.entries) != 1 {
		t.Fatal("a broken entry must not be dropped; a later undo would apply an older patch to a mismatched state")
	}
}
