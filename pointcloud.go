package main

import (
	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

type transformedVec3RandomAccessor struct {
	pc.Vec3RandomAccessor
	trans mat.Mat4
}

func (a *transformedVec3RandomAccessor) Vec3At(i int) mat.Vec3 {
	return a.trans.TransformAffine(a.Vec3RandomAccessor.Vec3At(i))
}

func vec3Min(a, b mat.Vec3) mat.Vec3 {
	var out mat.Vec3
	for i := range out {
		if a[i] < b[i] {
			out[i] = a[i]
		} else {
			out[i] = b[i]
		}
	}
	return out
}

func float32Min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func float32Max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

type rect struct {
	min, max mat.Vec3
}

func rectIntersection(a, b rect) rect {
	r := rect{
		min: mat.Vec3{
			float32Max(a.min[0], b.min[0]),
			float32Max(a.min[1], b.min[1]),
			float32Max(a.min[2], b.min[2]),
		},
		max: mat.Vec3{
			float32Min(a.max[0], b.max[0]),
			float32Min(a.max[1], b.max[1]),
			float32Min(a.max[2], b.max[2]),
		},
	}
	return r
}

func (r *rect) IsValid() bool {
	return !(r.min[0] > r.max[0] ||
		r.min[1] > r.max[1] ||
		r.min[2] > r.max[2])
}

func (r *rect) IsInside(v mat.Vec3) bool {
	return !(v[0] < r.min[0] ||
		v[1] < r.min[1] ||
		v[2] < r.min[2] ||
		r.max[0] < v[0] ||
		r.max[1] < v[1] ||
		r.max[2] < v[2])
}
