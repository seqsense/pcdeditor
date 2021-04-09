package main

import (
	"bytes"
	"syscall/js"

	"github.com/seqsense/pcgol/pc"
)

type pcdIOImpl struct{}

func (*pcdIOImpl) readPCD(path string) (*pc.PointCloud, error) {
	b, err := fetchGet(path)
	if err != nil {
		return nil, err
	}

	pp, err := pc.Unmarshal(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return pp, nil
}

func (*pcdIOImpl) exportPCD(pp *pc.PointCloud) (interface{}, error) {
	var buf bytes.Buffer
	if err := pc.Marshal(pp, &buf); err != nil {
		return nil, err
	}
	array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(array, buf.Bytes())

	blob := js.Global().Get("Blob").New([]interface{}{array}, map[string]interface{}{
		"type": "application.octet-stream",
	})
	return blob, nil
}
