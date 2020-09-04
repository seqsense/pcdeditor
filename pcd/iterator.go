package pcd

import (
	"encoding/binary"
	"math"

	"github.com/seqsense/pcdviewer/mat"
)

type binaryIterator struct {
	data   []byte
	pos    int
	stride int
}

func (i *binaryIterator) Incr() {
	i.pos += i.stride
}

func (i *binaryIterator) IsValid() bool {
	return i.pos+i.stride < len(i.data)
}

type Float32Iterator interface {
	Incr()
	IsValid() bool
	Float32() float32
}

type Vec3Iterator interface {
	Incr()
	IsValid() bool
	Vec3() mat.Vec3
}

type binaryFloat32Iterator struct {
	binaryIterator
}

func (i *binaryFloat32Iterator) Float32() float32 {
	return math.Float32frombits(
		binary.LittleEndian.Uint32(i.binaryIterator.data[i.binaryIterator.pos : i.binaryIterator.pos+4]),
	)
}

type float32Iterator struct {
	data   []float32
	pos    int
	stride int
}

func (i *float32Iterator) Incr() {
	i.pos += i.stride
}

func (i *float32Iterator) IsValid() bool {
	return i.pos+i.stride < len(i.data)
}

func (i *float32Iterator) Float32() float32 {
	return i.data[i.pos]
}

func (i *float32Iterator) Vec3() mat.Vec3 {
	return mat.Vec3{i.data[i.pos], i.data[i.pos+1], i.data[i.pos+2]}
}

type naiveVec3Iterator [3]Float32Iterator

func (i naiveVec3Iterator) IsValid() bool {
	return i[0].IsValid()
}

func (i naiveVec3Iterator) Incr() {
	i[0].Incr()
	i[1].Incr()
	i[2].Incr()
}
func (i naiveVec3Iterator) Vec3() mat.Vec3 {
	return mat.Vec3{i[0].Float32(), i[1].Float32(), i[2].Float32()}
}
