package gl

import (
	"reflect"
	"unsafe"
)

func float32SliceAsByteSlice(floats []float32) []byte {
	n := 4 * len(floats)

	up := unsafe.Pointer(&(floats[0]))
	pi := (*[1]byte)(up)
	buf := (*pi)[:]
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sh.Len = n
	sh.Cap = n

	return buf
}
