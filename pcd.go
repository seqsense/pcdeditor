package main

import (
	"bytes"
	"errors"
	"syscall/js"

	webgl "github.com/seqsense/pcdviewer/gl"
	"github.com/seqsense/pcdviewer/pcd"
)

func loadPCD(gl *webgl.WebGL, program webgl.Program, buf webgl.Buffer, path string) (*pcd.PointCloud, int, error) {
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
		return nil, 0, err
	}

	pc, err := pcd.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)

	return pc, pc.Points, nil
}
