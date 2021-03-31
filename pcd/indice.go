package pcd

import (
	"github.com/seqsense/pcdeditor/mat"
)

type indiceVec3RandomAccessor struct {
	indice []int
	ra     Vec3RandomAccessor
}

func (i *indiceVec3RandomAccessor) Len() int {
	return len(i.indice)
}

func (i *indiceVec3RandomAccessor) Vec3At(j int) mat.Vec3 {
	return i.ra.Vec3At(i.indice[j])
}

func NewIndiceVec3RandomAccessor(ra Vec3RandomAccessor, indice []int) Vec3RandomAccessor {
	return &indiceVec3RandomAccessor{
		ra:     ra,
		indice: indice,
	}
}
