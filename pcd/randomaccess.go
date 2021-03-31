package pcd

import (
	"github.com/seqsense/pcdeditor/mat"
)

type Vec3RandomAccessor interface {
	Vec3At(int) mat.Vec3
	Len() int
}
