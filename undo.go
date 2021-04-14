// +build !js

package main

import (
	"github.com/seqsense/pcgol/pc"
)

// historyDummy is a dummy history implementation for testing.
type historyDummy struct {
	latest *pc.PointCloud
}

func newHistory(_ int) history {
	return &historyDummy{}
}

func (historyDummy) MaxHistory() int {
	return 0
}

func (historyDummy) SetMaxHistory(_ int) {
}

func (h *historyDummy) push(pp *pc.PointCloud) *pc.PointCloud {
	h.latest = pp
	return pp
}

func (h *historyDummy) pop() *pc.PointCloud {
	return h.latest
}

func (historyDummy) undo() (*pc.PointCloud, bool) {
	return nil, false
}
