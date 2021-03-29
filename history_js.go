package main

import (
	"syscall/js"

	"github.com/seqsense/pcdeditor/pcd"
)

type historyJS struct {
	history       []js.Value
	historyHeader []pcd.PointCloudHeader
	maxHistory    int
}

func newHistory(n int) history {
	return &historyJS{maxHistory: n}
}

func (h *historyJS) MaxHistory() int {
	return h.maxHistory
}

func (h *historyJS) SetMaxHistory(m int) {
	if m < 0 {
		m = 0
	}
	h.maxHistory = m
}

func (h *historyJS) push(pc *pcd.PointCloud) *pcd.PointCloud {
	header := pc.PointCloudHeader.Clone()
	dataJS := js.Global().Get("Uint8Array").New(len(pc.Data))
	js.CopyBytesToJS(dataJS, pc.Data)
	h.history = append(h.history, dataJS)
	h.historyHeader = append(h.historyHeader, header)
	if len(h.history) > h.MaxHistory()+1 {
		h.history = h.history[1:]
		h.historyHeader = h.historyHeader[1:]
	}
	return pc
}

func (h *historyJS) pop() *pcd.PointCloud {
	back := h.history[len(h.history)-1]
	backHeader := h.historyHeader[len(h.historyHeader)-1]
	h.history = h.history[:len(h.history)-1]
	h.historyHeader = h.historyHeader[:len(h.historyHeader)-1]

	return h.reconstructPointCloud(backHeader, back)
}

func (h *historyJS) undo() (*pcd.PointCloud, bool) {
	if n := len(h.history); n > 1 {
		h.history = h.history[:n-1]
		h.historyHeader = h.historyHeader[:n-1]

		return h.reconstructPointCloud(h.historyHeader[n-2], h.history[n-2]), true
	}
	return nil, false
}

func (h *historyJS) reconstructPointCloud(header pcd.PointCloudHeader, dataJS js.Value) *pcd.PointCloud {
	pc := &pcd.PointCloud{
		PointCloudHeader: header,
		Points:           header.Width * header.Height,
		Data:             make([]byte, dataJS.Get("byteLength").Int()),
	}
	js.CopyBytesToGo(pc.Data, dataJS)
	return pc
}
