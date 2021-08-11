package main

import (
	"syscall/js"

	"github.com/seqsense/pcgol/pc"
)

type historyJS struct {
	history       []js.Value
	historyHeader []pc.PointCloudHeader
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

func (h *historyJS) push(pp *pc.PointCloud) *pc.PointCloud {
	header := pp.PointCloudHeader.Clone()
	dataJS := js.Global().Get("Uint8Array").New(len(pp.Data))
	js.CopyBytesToJS(dataJS, pp.Data)
	h.history = append(h.history, dataJS)
	h.historyHeader = append(h.historyHeader, header)
	if len(h.history) > h.MaxHistory()+1 {
		h.history[0] = js.Null()
		h.history = h.history[1:]
		h.historyHeader = h.historyHeader[1:]
	}
	return pp
}

func (h *historyJS) pop() *pc.PointCloud {
	n := len(h.history)
	back := h.history[n-1]
	backHeader := h.historyHeader[n-1]
	h.history[n-1] = js.Null()
	h.history = h.history[:n-1]
	h.historyHeader = h.historyHeader[:n-1]

	return h.reconstructPointCloud(backHeader, back)
}

func (h *historyJS) undo() (*pc.PointCloud, bool) {
	if n := len(h.history); n > 1 {
		h.history[n-1] = js.Null()
		h.history = h.history[:n-1]
		h.historyHeader = h.historyHeader[:n-1]

		return h.reconstructPointCloud(h.historyHeader[n-2], h.history[n-2]), true
	}
	return nil, false
}

func (h *historyJS) reconstructPointCloud(header pc.PointCloudHeader, dataJS js.Value) *pc.PointCloud {
	pp := &pc.PointCloud{
		PointCloudHeader: header,
		Points:           header.Width * header.Height,
		Data:             make([]byte, dataJS.Get("byteLength").Int()),
	}
	js.CopyBytesToGo(pp.Data, dataJS)
	return pp
}
