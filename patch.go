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
	// The wire form is the head followed by the raw payload
	encodeHead(buf *bytes.Buffer)
	payload() []byte
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
)

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

func (p *replacePatch) encodeHead(buf *bytes.Buffer) {
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
}

func (p *replacePatch) payload() []byte {
	return p.data
}

func encodePatches(buf *bytes.Buffer, ps []patch) {
	for _, p := range ps {
		p.encodeHead(buf)
		buf.Write(p.payload())
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
	case patchTypeReplace:
		p := &replacePatch{}
		p.header.Version = math.Float32frombits(r.uint32())
		nFields := int(r.uint32())
		// A field encodes to at least 16 bytes
		if r.err != nil || nFields < 0 || nFields > len(r.b)/16 {
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
		if r.err != nil || nvp < 0 || nvp > len(r.b)/4 {
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
	p.encodeHead(&buf)
	data := p.payload()
	packed := make([]byte, buf.Len()+len(data))
	copy(packed, buf.Bytes())
	copy(packed[buf.Len():], data)
	return packed
}

func writeUint32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
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
