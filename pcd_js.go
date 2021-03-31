package main

import (
	"bytes"
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
