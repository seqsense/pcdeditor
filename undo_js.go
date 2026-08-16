package main

import (
	"syscall/js"

	"github.com/seqsense/pcgol/pc"
)

// historyJS stores entries as JS Uint8Arrays to keep them out of the WASM
// linear memory, which never shrinks.
type historyJS struct {
	// entries[i] is a list of packed patch chunks forming one undo step
	entries    [][]js.Value
	maxHistory int
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

func (h *historyJS) push(p patch) {
	packed := packPatch(p)
	chunk := js.Global().Get("Uint8Array").New(len(packed))
	js.CopyBytesToJS(chunk, packed)
	h.entries = append(h.entries, []js.Value{chunk})
	for len(h.entries) > h.maxHistory {
		h.entries[0] = nil
		h.entries = h.entries[1:]
	}
}

func (h *historyJS) squashLatest() {
	if n := len(h.entries); n >= 2 {
		h.entries[n-2] = append(h.entries[n-2], h.entries[n-1]...)
		h.entries[n-1] = nil
		h.entries = h.entries[:n-1]
	}
}

func (h *historyJS) undo(pp *pc.PointCloud) (*pc.PointCloud, bool) {
	n := len(h.entries)
	if n == 0 {
		return nil, false
	}
	entry := h.entries[n-1]
	h.entries[n-1] = nil
	h.entries = h.entries[:n-1]

	chunks := make([][]byte, len(entry))
	for i, c := range entry {
		b := make([]byte, c.Get("byteLength").Int())
		js.CopyBytesToGo(b, c)
		chunks[i] = b
	}
	out, err := revertChunks(pp, chunks)
	if err != nil {
		return nil, false
	}
	return out, true
}

func (h *historyJS) clear() {
	h.entries = nil
}
