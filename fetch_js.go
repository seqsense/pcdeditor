package main

import (
	"errors"
	"fmt"
	"syscall/js"
)

func fetchGet(path string) ([]byte, error) {
	var b []byte
	var errored bool
	chErr := make(chan error)
	js.Global().Call("fetch", path, map[string]interface{}{
		"credentials": "include",
	}).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if !args[0].Get("ok").Bool() {
				chErr <- fmt.Errorf("failed to fetch file: %s", args[0].Get("statusText").String())
				errored = true
				return nil
			}
			return args[0].Call("arrayBuffer")
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to fetch file")
			errored = true
			return nil
		}),
	).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if errored {
				return nil
			}
			array := js.Global().Get("Uint8Array").New(args[0])
			n := array.Get("byteLength").Int()
			b = make([]byte, n)
			js.CopyBytesToGo(b, array)
			chErr <- nil
			return nil
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to handle received data")
			return nil
		}),
	)

	if err := <-chErr; err != nil {
		return nil, err
	}

	return b, nil
}
