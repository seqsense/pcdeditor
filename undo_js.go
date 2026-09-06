package main

import (
	"syscall/js"
)

func newHistory(n int) *history {
	return &history{store: jsStore{}, maxHistory: n}
}

// jsStore keeps records on the JS heap to avoid growing the WASM linear memory.
type jsStore struct{}

func (jsStore) store(parts ...[]byte) record {
	var total int
	for _, d := range parts {
		total += len(d)
	}
	v := js.Global().Get("Uint8Array").New(total)
	var off int
	for _, d := range parts {
		js.CopyBytesToJS(v.Call("subarray", off), d)
		off += len(d)
	}
	return jsRecord{v}
}

type jsRecord struct {
	v js.Value
}

func (r jsRecord) load() []byte {
	b := make([]byte, r.v.Get("byteLength").Int())
	js.CopyBytesToGo(b, r.v)
	return b
}
