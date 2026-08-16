package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"

	"github.com/seqsense/pcgol/pc"
)

type patch interface {
	// pp may be mutated; use the returned cloud
	revert(pp *pc.PointCloud) (*pc.PointCloud, error)
	encode(buf *bytes.Buffer)
}

const (
	patchTypeLabel = iota + 1
	patchTypeDelete
	patchTypeAppend
	patchTypeReplace
)

var (
	errBrokenPatch      = errors.New("broken patch data")
	errUnknownPatchType = errors.New("unknown patch type")
	errNoLabelField     = errors.New("point cloud has no label field")
)

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

type labelPatch struct {
	indices   []uint32
	oldLabels []uint32
}

func (p *labelPatch) revert(pp *pc.PointCloud) (*pc.PointCloud, error) {
	off, ok := fieldByteOffset(&pp.PointCloudHeader, "label")
	if !ok {
		return nil, errNoLabelField
	}
	stride := pp.Stride()
	for k, idx := range p.indices {
		i := int(idx)*stride + off
		if i+4 > len(pp.Data) {
			return nil, errBrokenPatch
		}
		binary.LittleEndian.PutUint32(pp.Data[i:], p.oldLabels[k])
	}
	return pp, nil
}

func (p *labelPatch) encode(buf *bytes.Buffer) {
	buf.WriteByte(patchTypeLabel)
	writeUint32(buf, uint32(len(p.indices)))
	writeUint32s(buf, p.indices)
	writeUint32s(buf, p.oldLabels)
}

type deletePatch struct {
	oldWidth, oldHeight int
	indices             []uint32 // ascending original positions of the removed points
	points              []byte
}

func (p *deletePatch) revert(pp *pc.PointCloud) (*pc.PointCloud, error) {
	stride := pp.Stride()
	if len(p.points) != len(p.indices)*stride {
		return nil, errBrokenPatch
	}
	oldN := pp.Points + len(p.indices)
	need := oldN * stride
	var data []byte
	if cap(pp.Data) >= need {
		data = pp.Data[:need]
	} else {
		data = make([]byte, need)
		copy(data, pp.Data)
	}

	// Walk backwards so that every move reads a not-yet-overwritten position
	di := len(p.indices) - 1
	src := pp.Points - 1
	for dst := oldN - 1; dst >= 0; dst-- {
		if di >= 0 && int(p.indices[di]) == dst {
			copy(data[dst*stride:(dst+1)*stride], p.points[di*stride:(di+1)*stride])
			di--
		} else {
			if src < 0 {
				return nil, errBrokenPatch
			}
			if dst != src {
				copy(data[dst*stride:(dst+1)*stride], data[src*stride:(src+1)*stride])
			}
			src--
		}
	}
	if di >= 0 {
		return nil, errBrokenPatch
	}
	return newCloudView(pp, oldN, p.oldWidth, p.oldHeight, data), nil
}

func (p *deletePatch) encode(buf *bytes.Buffer) {
	buf.WriteByte(patchTypeDelete)
	writeUint32(buf, uint32(p.oldWidth))
	writeUint32(buf, uint32(p.oldHeight))
	writeUint32(buf, uint32(len(p.indices)))
	writeUint32s(buf, p.indices)
	writeUint32(buf, uint32(len(p.points)))
	buf.Write(p.points)
}

type appendPatch struct {
	oldPoints, oldWidth, oldHeight int
}

func (p *appendPatch) revert(pp *pc.PointCloud) (*pc.PointCloud, error) {
	stride := pp.Stride()
	if p.oldPoints > pp.Points || p.oldPoints*stride > len(pp.Data) {
		return nil, errBrokenPatch
	}
	return newCloudView(pp, p.oldPoints, p.oldWidth, p.oldHeight, pp.Data[:p.oldPoints*stride]), nil
}

func (p *appendPatch) encode(buf *bytes.Buffer) {
	buf.WriteByte(patchTypeAppend)
	writeUint32(buf, uint32(p.oldPoints))
	writeUint32(buf, uint32(p.oldWidth))
	writeUint32(buf, uint32(p.oldHeight))
}

type replacePatch struct {
	header pc.PointCloudHeader
	data   []byte
}

func (p *replacePatch) revert(_ *pc.PointCloud) (*pc.PointCloud, error) {
	return &pc.PointCloud{
		PointCloudHeader: p.header,
		Points:           p.header.Width * p.header.Height,
		Data:             p.data,
	}, nil
}

func (p *replacePatch) encode(buf *bytes.Buffer) {
	buf.WriteByte(patchTypeReplace)
	writeUint32(buf, math.Float32bits(p.header.Version))
	writeUint32(buf, uint32(len(p.header.Fields)))
	for i := range p.header.Fields {
		writeString(buf, p.header.Fields[i])
		writeUint32(buf, uint32(p.header.Size[i]))
		writeString(buf, p.header.Type[i])
		writeUint32(buf, uint32(p.header.Count[i]))
	}
	writeUint32(buf, uint32(p.header.Width))
	writeUint32(buf, uint32(p.header.Height))
	writeUint32(buf, uint32(len(p.header.Viewpoint)))
	for _, v := range p.header.Viewpoint {
		writeUint32(buf, math.Float32bits(v))
	}
	writeUint32(buf, uint32(len(p.data)))
	buf.Write(p.data)
}

func encodePatches(buf *bytes.Buffer, ps []patch) {
	for _, p := range ps {
		p.encode(buf)
	}
}

// Decoded patches may reference b; do not reuse it afterwards
func decodePatches(b []byte) ([]patch, error) {
	var ps []patch
	for len(b) > 0 {
		p, rest, err := decodePatch(b)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
		b = rest
	}
	return ps, nil
}

func decodePatch(b []byte) (patch, []byte, error) {
	if len(b) < 1 {
		return nil, nil, errBrokenPatch
	}
	typ := b[0]
	r := reader{b: b[1:]}
	switch typ {
	case patchTypeLabel:
		n := int(r.uint32())
		p := &labelPatch{
			indices:   r.uint32s(n),
			oldLabels: r.uint32s(n),
		}
		if r.err != nil {
			return nil, nil, r.err
		}
		return p, r.b, nil
	case patchTypeDelete:
		p := &deletePatch{
			oldWidth:  int(r.uint32()),
			oldHeight: int(r.uint32()),
		}
		p.indices = r.uint32s(int(r.uint32()))
		p.points = r.bytes(int(r.uint32()))
		if r.err != nil {
			return nil, nil, r.err
		}
		return p, r.b, nil
	case patchTypeAppend:
		p := &appendPatch{
			oldPoints: int(r.uint32()),
			oldWidth:  int(r.uint32()),
			oldHeight: int(r.uint32()),
		}
		if r.err != nil {
			return nil, nil, r.err
		}
		return p, r.b, nil
	case patchTypeReplace:
		p := &replacePatch{}
		p.header.Version = math.Float32frombits(r.uint32())
		nFields := int(r.uint32())
		if r.err != nil || nFields < 0 || nFields > len(r.b) {
			return nil, nil, errBrokenPatch
		}
		p.header.Fields = make([]string, nFields)
		p.header.Size = make([]int, nFields)
		p.header.Type = make([]string, nFields)
		p.header.Count = make([]int, nFields)
		for i := 0; i < nFields; i++ {
			p.header.Fields[i] = r.string()
			p.header.Size[i] = int(r.uint32())
			p.header.Type[i] = r.string()
			p.header.Count[i] = int(r.uint32())
		}
		p.header.Width = int(r.uint32())
		p.header.Height = int(r.uint32())
		nvp := int(r.uint32())
		if r.err != nil || nvp < 0 || nvp*4 > len(r.b) {
			return nil, nil, errBrokenPatch
		}
		p.header.Viewpoint = make([]float32, nvp)
		for i := range p.header.Viewpoint {
			p.header.Viewpoint[i] = math.Float32frombits(r.uint32())
		}
		p.data = r.bytes(int(r.uint32()))
		if r.err != nil {
			return nil, nil, r.err
		}
		return p, r.b, nil
	}
	return nil, nil, errUnknownPatchType
}

func revertChunks(pp *pc.PointCloud, chunks [][]byte) (*pc.PointCloud, error) {
	for i := len(chunks) - 1; i >= 0; i-- {
		ps, err := decodePatches(chunks[i])
		if err != nil {
			return nil, err
		}
		for j := len(ps) - 1; j >= 0; j-- {
			if pp, err = ps[j].revert(pp); err != nil {
				return nil, err
			}
		}
	}
	return pp, nil
}

func packPatch(p patch) []byte {
	var buf bytes.Buffer
	p.encode(&buf)
	return buf.Bytes()
}

func writeUint32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func writeUint32s(buf *bytes.Buffer, vs []uint32) {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	buf.Write(b)
}

func writeString(buf *bytes.Buffer, s string) {
	writeUint32(buf, uint32(len(s)))
	buf.WriteString(s)
}

type reader struct {
	b   []byte
	err error
}

func (r *reader) uint32() uint32 {
	if r.err != nil {
		return 0
	}
	if len(r.b) < 4 {
		r.err = errBrokenPatch
		return 0
	}
	v := binary.LittleEndian.Uint32(r.b)
	r.b = r.b[4:]
	return v
}

func (r *reader) uint32s(n int) []uint32 {
	if r.err != nil {
		return nil
	}
	if n < 0 || len(r.b) < 4*n {
		r.err = errBrokenPatch
		return nil
	}
	vs := make([]uint32, n)
	for i := range vs {
		vs[i] = binary.LittleEndian.Uint32(r.b[i*4:])
	}
	r.b = r.b[4*n:]
	return vs
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || len(r.b) < n {
		r.err = errBrokenPatch
		return nil
	}
	b := r.b[:n]
	r.b = r.b[n:]
	return b
}

func (r *reader) string() string {
	return string(r.bytes(int(r.uint32())))
}
