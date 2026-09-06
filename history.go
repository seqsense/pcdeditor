package main

import (
	"bytes"

	"github.com/seqsense/pcgol/pc"
)

type recordStore interface {
	store(parts ...[]byte) record
}

type record interface {
	load() []byte
}

type undoStep []record

type history struct {
	store      recordStore
	steps      []undoStep
	maxHistory int
}

func (h *history) MaxHistory() int {
	return h.maxHistory
}

func (h *history) SetMaxHistory(m int) {
	if m < 0 {
		m = 0
	}
	h.maxHistory = m
}

func (h *history) push(d undoData) error {
	var head bytes.Buffer
	if err := encodeUndoData(&head, d); err != nil {
		return err
	}
	h.steps = append(h.steps, undoStep{h.store.store(head.Bytes(), d.payload())})
	for len(h.steps) > h.maxHistory {
		h.steps[0] = nil
		h.steps = h.steps[1:]
	}
	return nil
}

func (h *history) squashLatest() {
	if n := len(h.steps); n >= 2 {
		h.steps[n-2] = append(h.steps[n-2], h.steps[n-1]...)
		h.steps[n-1] = nil
		h.steps = h.steps[:n-1]
	}
}

func (h *history) undo(pp *pc.PointCloud) (*pc.PointCloud, bool) {
	n := len(h.steps)
	if n == 0 {
		return nil, false
	}
	step := h.steps[n-1]
	for i := len(step) - 1; i >= 0; i-- {
		d, err := decodeRecord(step[i].load())
		if err != nil {
			return nil, false
		}
		if pp, err = d.restore(pp); err != nil {
			return nil, false
		}
	}
	h.steps[n-1] = nil
	h.steps = h.steps[:n-1]
	return pp, true
}

func (h *history) clear() {
	h.steps = nil
}
