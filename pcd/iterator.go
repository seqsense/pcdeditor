package pcd

import (
	"encoding/binary"
	"math"
)

type Iterator struct {
	data   []byte
	pos    int
	stride int
}

func (i *Iterator) Incr() {
	i.pos += i.stride
}

func (i *Iterator) IsValid() bool {
	return i.pos+i.stride < len(i.data)
}

type Float32Iterator struct {
	Iterator
}

func (i *Float32Iterator) Float32() float32 {
	return math.Float32frombits(
		binary.LittleEndian.Uint32(i.Iterator.data[i.Iterator.pos : i.Iterator.pos+4]),
	)
}
