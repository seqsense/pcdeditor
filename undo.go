//go:build !js
// +build !js

package main

import (
	"github.com/seqsense/pcgol/pc"
)

type historyMem struct {
	// entries[i] is a list of packed patch chunks forming one undo step
	entries    [][][]byte
	maxHistory int
}

func newHistory(n int) history {
	return &historyMem{maxHistory: n}
}

func (h *historyMem) MaxHistory() int {
	return h.maxHistory
}

func (h *historyMem) SetMaxHistory(m int) {
	if m < 0 {
		m = 0
	}
	h.maxHistory = m
}

func (h *historyMem) push(p patch) {
	h.entries = append(h.entries, [][]byte{packPatch(p)})
	for len(h.entries) > h.maxHistory {
		h.entries[0] = nil
		h.entries = h.entries[1:]
	}
}

func (h *historyMem) squashLatest() {
	if n := len(h.entries); n >= 2 {
		h.entries[n-2] = append(h.entries[n-2], h.entries[n-1]...)
		h.entries[n-1] = nil
		h.entries = h.entries[:n-1]
	}
}

func (h *historyMem) undo(pp *pc.PointCloud) (*pc.PointCloud, bool) {
	n := len(h.entries)
	if n == 0 {
		return nil, false
	}
	entry := h.entries[n-1]
	h.entries[n-1] = nil
	h.entries = h.entries[:n-1]

	out, err := revertChunks(pp, entry)
	if err != nil {
		return nil, false
	}
	return out, true
}

func (h *historyMem) clear() {
	h.entries = nil
}
