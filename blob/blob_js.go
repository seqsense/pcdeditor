package blob

import (
	"errors"
	"io"
	"syscall/js"
)

type Blob js.Value

var blobJS = js.Global().Get("Blob")

func New(b []byte, typ string) Blob {
	array := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(array, b)

	return Blob(blobJS.New([]interface{}{array}, map[string]interface{}{
		"type": typ,
	}))
}

func JS(j interface{}) (Blob, error) {
	jv, ok := j.(js.Value)
	if !ok {
		return Blob{}, errors.New("requires JavaScript object")
	}
	if !jv.InstanceOf(blobJS) {
		return Blob{}, errors.New("requires Blob object")
	}
	return Blob(jv), nil
}

func (blob Blob) JS() js.Value {
	return js.Value(blob)
}

func (blob Blob) Reader() (io.Reader, error) {
	var r *blobReader
	chErr := make(chan error)
	js.Value(blob).Call("arrayBuffer").Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			array := js.Global().Get("Uint8Array").New(args[0])
			n := array.Get("byteLength").Int()
			r = &blobReader{
				jsArray: array,
				n:       n,
			}
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

	return r, nil
}

type blobReader struct {
	jsArray js.Value
	n       int
	pos     int
}

func (r *blobReader) Read(b []byte) (int, error) {
	if r.n == r.pos {
		return 0, io.EOF
	}
	end := r.pos + len(b)
	if end > r.n {
		end = r.n
	}
	n := end - r.pos
	sa := r.jsArray.Call("subarray", js.ValueOf(r.pos), js.ValueOf(end))
	js.CopyBytesToGo(b, sa)
	r.pos = end
	return n, nil
}
