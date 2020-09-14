package main

import (
	"bytes"
	"errors"
	"syscall/js"

	"github.com/seqsense/pcdeditor/pcd"
)

type pcdIOImpl struct{}

func (*pcdIOImpl) readPCD(path string) (*pcd.PointCloud, error) {
	b, err := fetchGet(path)
	if err != nil {
		return nil, err
	}

	pc, err := pcd.Unmarshal(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return pc, nil
}

func (*pcdIOImpl) writePCD(path string, pc *pcd.PointCloud) error {
	var buf bytes.Buffer
	if err := pcd.Marshal(pc, &buf); err != nil {
		return err
	}
	array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(array, buf.Bytes())

	chErr := make(chan error)
	js.Global().Call("fetch", path, map[string]interface{}{
		"method":      "PUT",
		"headers":     map[string]interface{}{"Content-Type": "application/octet-stream"},
		"credentials": "include",
		"body":        array,
	}).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- nil
			return nil
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to fetch file")
			return nil
		}),
	)

	if err := <-chErr; err != nil {
		return err
	}

	return nil
}

func (*pcdIOImpl) exportPCD(pc *pcd.PointCloud) (interface{}, error) {
	var buf bytes.Buffer
	if err := pcd.Marshal(pc, &buf); err != nil {
		return nil, err
	}
	array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(array, buf.Bytes())

	blob := js.Global().Get("Blob").New([]interface{}{array}, map[string]interface{}{
		"type": "application.octet-stream",
	})
	return blob, nil
}
