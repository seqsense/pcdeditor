//go:build !js
// +build !js

package main

func newHistory(n int) *history {
	return &history{store: memStore{}, maxHistory: n}
}

// memStore keeps records on the Go heap.
type memStore struct{}

func (memStore) store(parts ...[]byte) record {
	var total int
	for _, d := range parts {
		total += len(d)
	}
	b := make([]byte, 0, total)
	for _, d := range parts {
		b = append(b, d...)
	}
	return memRecord(b)
}

type memRecord []byte

func (r memRecord) load() []byte {
	return r
}
