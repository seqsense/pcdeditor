package gl

import (
	"reflect"
	"unsafe"
)

func byteSliceAsUInt32Slice(bytes []byte) []uint32 {
	n := len(bytes) / 4

	up := unsafe.Pointer(&(bytes[0]))
	pi := (*[1]uint32)(up)
	buf := (*pi)[:]
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sh.Len = n
	sh.Cap = n

	return buf
}
