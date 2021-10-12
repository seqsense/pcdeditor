package blob

import (
	"errors"
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

func (blob Blob) Bytes() ([]byte, error) {
	var b []byte
	chErr := make(chan error)
	js.Value(blob).Call("arrayBuffer").Call("then",
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

	return b, nil
}
