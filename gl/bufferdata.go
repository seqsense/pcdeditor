package gl

type BufferData interface {
	Bytes() []byte
}

type Float32ArrayBuffer []float32

func (b Float32ArrayBuffer) Bytes() []byte {
	return float32SliceAsByteSlice([]float32(b))
}

type ByteArrayBuffer []byte

func (b ByteArrayBuffer) Bytes() []byte {
	return b
}
