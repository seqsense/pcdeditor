package pcd

import (
	"reflect"
	"unsafe"
)

func byteSliceAsFloat32Slice(b []byte) []float32 {
	n := len(b) / 4

	up := unsafe.Pointer(&(b[0]))
	pi := (*[1]float32)(up)
	buf := (*pi)[:]
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sh.Len = n
	sh.Cap = n

	return buf
}

func isShadowing(b []byte, f []float32) bool {
	return uintptr(unsafe.Pointer(&f[0])) != uintptr(unsafe.Pointer(&b[0]))
}
