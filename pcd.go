package main

import (
	"bytes"
	"errors"
	"syscall/js"

	"github.com/seqsense/pcdviewer/pcd"
)

func readPCD(path string) (*pcd.PointCloud, error) {
	var b []byte
	chErr := make(chan error)
	js.Global().Call("fetch", path).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return args[0].Call("arrayBuffer")
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to fetch file")
			return nil
		}),
	).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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

	pc, err := pcd.Unmarshal(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return pc, nil
}

func writePCD(path string, pc *pcd.PointCloud) error {
	var buf bytes.Buffer
	if err := pcd.Marshal(pc, &buf); err != nil {
		return err
	}
	array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(array, buf.Bytes())

	chErr := make(chan error)
	js.Global().Call("fetch", path, map[string]interface{}{
		"method":  "POST",
		"headers": map[string]interface{}{"Content-Type": "application/octet-stream"},
		"body":    array,
	}).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return args[0].Call("arrayBuffer")
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to fetch file")
			return nil
		}),
	).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- nil
			return nil
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to handle received data")
			return nil
		}),
	)

	if err := <-chErr; err != nil {
		return err
	}

	return nil
}
