package main

import (
	"bytes"

	"github.com/seqsense/pcdeditor/blob"
	"github.com/seqsense/pcgol/pc"
)

type pcdIOImpl struct{}

func (*pcdIOImpl) importPCD(b interface{}) (*pc.PointCloud, error) {
	bj, err := blob.JS(b)
	if err != nil {
		return nil, err
	}
	bs, err := bj.Bytes()
	if err != nil {
		return nil, err
	}
	pp, err := pc.Unmarshal(bytes.NewReader(bs))
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
	return blob.New(buf.Bytes(), "application/x-pcd").JS(), nil
}
