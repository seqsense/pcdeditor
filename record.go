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
	gob.Register(&undoDataSize{})
	gob.Register(&undoDataRemovedPoints{})
}

// pcgol caches an unsafe float32 alias of Data keyed only by its base pointer,
// so a change of the Data length must be delivered in a fresh PointCloud.
func newCloudView(pp *pc.PointCloud, points, width, height int, data []byte) *pc.PointCloud {
	out := &pc.PointCloud{
		PointCloudHeader: pp.PointCloudHeader.Clone(),
		Points:           points,
		Data:             data,
	}
	out.Width = width
	out.Height = height
	return out
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

type undoDataRemovedPoints struct {
	OldWidth, OldHeight int
	Indices             []uint32 // ascending original positions of the removed points
	points              []byte
}

func (p *undoDataRemovedPoints) restore(pp *pc.PointCloud) (*pc.PointCloud, error) {
	stride := pp.Stride()
	if len(p.points) != len(p.Indices)*stride {
		return nil, errBrokenRecord
	}
	oldN := pp.Points + len(p.Indices)
	need := oldN * stride
	var data []byte
	if cap(pp.Data) >= need {
		data = pp.Data[:need]
	} else {
		data = make([]byte, need)
		copy(data, pp.Data)
	}

	// Walk backwards so that every move reads a not-yet-overwritten position
	di := len(p.Indices) - 1
	src := pp.Points - 1
	for dst := oldN - 1; dst >= 0; dst-- {
		if di >= 0 && int(p.Indices[di]) == dst {
			copy(data[dst*stride:(dst+1)*stride], p.points[di*stride:(di+1)*stride])
			di--
		} else {
			if src < 0 {
				return nil, errBrokenRecord
			}
			if dst != src {
				copy(data[dst*stride:(dst+1)*stride], data[src*stride:(src+1)*stride])
			}
			src--
		}
	}
	if di >= 0 {
		return nil, errBrokenRecord
	}
	return newCloudView(pp, oldN, p.OldWidth, p.OldHeight, data), nil
}

func (p *undoDataRemovedPoints) payload() []byte {
	return p.points
}

func (p *undoDataRemovedPoints) setPayload(data []byte) {
	p.points = data
}

type undoDataSize struct {
	Points, Width, Height int
}

func (p *undoDataSize) restore(pp *pc.PointCloud) (*pc.PointCloud, error) {
	stride := pp.Stride()
	if stride <= 0 || p.Points < 0 || p.Width < 0 || p.Height < 0 ||
		p.Points > pp.Points || p.Points > len(pp.Data)/stride {
		return nil, errBrokenRecord
	}
	return newCloudView(pp, p.Points, p.Width, p.Height, pp.Data[:p.Points*stride]), nil
}

func (p *undoDataSize) payload() []byte {
	return nil
}

func (p *undoDataSize) setPayload([]byte) {}

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
