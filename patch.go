package main

import (
	"bytes"
	"encoding/gob"
	"io"

	"github.com/seqsense/pcgol/pc"
)

// patch is a reverse edit restoring a point cloud to the state before an edit.
type patch interface {
	// pp may be mutated; use the returned cloud.
	revert(pp *pc.PointCloud) (*pc.PointCloud, error)
	// payload returns the bulk data kept out of the gob encoding to avoid copying it.
	payload() []byte
	setPayload(data []byte)
}

func init() {
	gob.Register(&replacePatch{})
}

type replacePatch struct {
	Header pc.PointCloudHeader
	data   []byte
}

func (p *replacePatch) revert(_ *pc.PointCloud) (*pc.PointCloud, error) {
	return &pc.PointCloud{
		PointCloudHeader: p.Header,
		Points:           p.Header.Width * p.Header.Height,
		Data:             p.data,
	}, nil
}

func (p *replacePatch) payload() []byte {
	return p.data
}

func (p *replacePatch) setPayload(data []byte) {
	p.data = data
}

// encodePatch writes the gob encoding of p to w, excluding the payload.
// A complete record is this encoding followed by the raw payload.
func encodePatch(w io.Writer, p patch) error {
	return gob.NewEncoder(w).Encode(&p)
}

// The returned patch references b; do not reuse b afterwards.
func decodePatch(b []byte) (patch, error) {
	r := bytes.NewReader(b)
	var p patch
	if err := gob.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}
	// gob reads exactly one value from a ByteReader; the rest is the payload
	p.setPayload(b[len(b)-r.Len():])
	return p, nil
}
