// +build !js

package main

import (
	"github.com/seqsense/pcdeditor/pcd"
)

// historyDummy is a dummy history implementation for testing.
type historyDummy struct {
	latest *pcd.PointCloud
}

func newHistory(_ int) history {
	return &historyDummy{}
}

func (historyDummy) MaxHistory() int {
	return 0
}

func (historyDummy) SetMaxHistory(_ int) {
}

func (h *historyDummy) push(pc *pcd.PointCloud) *pcd.PointCloud {
	h.latest = pc
	return pc
}

func (h *historyDummy) pop() *pcd.PointCloud {
	return h.latest
}

func (historyDummy) undo() (*pcd.PointCloud, bool) {
	return nil, false
}
