package pcd

import (
	"encoding/binary"
	"math"
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
