package main

import (
	"bytes"
	"encoding/gob"
	"io"

	"github.com/seqsense/pcgol/pc"
)

type undoData interface {
	// pp may be mutated; use the returned cloud.
	restore(pp *pc.PointCloud) (*pc.PointCloud, error)
	// payload returns the bulk data kept out of the gob encoding to avoid copying it.
	payload() []byte
	setPayload(data []byte)
}

func init() {
	gob.Register(&undoDataEntireCloud{})
}

type undoDataEntireCloud struct {
	Header pc.PointCloudHeader
	data   []byte
}

func newUndoDataEntireCloud(pp *pc.PointCloud) *undoDataEntireCloud {
	return &undoDataEntireCloud{
		Header: pp.PointCloudHeader.Clone(),
		data:   pp.Data,
	}
}

func (p *undoDataEntireCloud) restore(_ *pc.PointCloud) (*pc.PointCloud, error) {
	return &pc.PointCloud{
		PointCloudHeader: p.Header,
		Points:           p.Header.Width * p.Header.Height,
		Data:             p.data,
	}, nil
}

func (p *undoDataEntireCloud) payload() []byte {
	return p.data
}

func (p *undoDataEntireCloud) setPayload(data []byte) {
	p.data = data
}

// A record is this encoding followed by the raw payload.
func encodeUndoData(w io.Writer, d undoData) error {
	return gob.NewEncoder(w).Encode(&d)
}

// The returned undoData references b; do not reuse b afterwards.
func decodeRecord(b []byte) (undoData, error) {
	r := bytes.NewReader(b)
	var d undoData
	if err := gob.NewDecoder(r).Decode(&d); err != nil {
		return nil, err
	}
	// gob reads exactly one value from a ByteReader; the rest is the payload
	d.setPayload(b[len(b)-r.Len():])
	return d, nil
}
