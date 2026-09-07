package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
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
	gob.Register(&undoDataLabels{})
}

var (
	errBrokenRecord = errors.New("broken undo record")
	errNoLabelField = errors.New("point cloud has no label field")
)

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

func fieldByteOffset(h *pc.PointCloudHeader, name string) (int, bool) {
	offset := 0
	for i, fn := range h.Fields {
		if fn == name {
			return offset, true
		}
		offset += h.Size[i] * h.Count[i]
	}
	return 0, false
}

type undoDataLabels struct {
	Indices   []uint32
	OldLabels []uint32
}

func (p *undoDataLabels) restore(pp *pc.PointCloud) (*pc.PointCloud, error) {
	if len(p.Indices) != len(p.OldLabels) {
		return nil, errBrokenRecord
	}
	off, ok := fieldByteOffset(&pp.PointCloudHeader, "label")
	if !ok {
		return nil, errNoLabelField
	}
	stride := pp.Stride()
	for k, idx := range p.Indices {
		i := int(idx)*stride + off
		if i+4 > len(pp.Data) {
			return nil, errBrokenRecord
		}
		binary.LittleEndian.PutUint32(pp.Data[i:], p.OldLabels[k])
	}
	return pp, nil
}

func (p *undoDataLabels) payload() []byte {
	return nil
}

func (p *undoDataLabels) setPayload([]byte) {}

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
