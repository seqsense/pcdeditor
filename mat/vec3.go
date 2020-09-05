package mat

import (
	"math"
)

type Vec3 [3]float32

func NewVec3(x, y, z float32) Vec3 {
	return Vec3{x, y, z}
}

func (v Vec3) NormSq() float32 {
	return v[0]*v[0] + v[1]*v[1] + v[2]*v[2]
}

func (v Vec3) Norm() float32 {
	return float32(math.Sqrt(float64(v.NormSq())))
}

func (v Vec3) Normalized() Vec3 {
	return v.Mul(1.0 / v.Norm())
}

func (v Vec3) Mul(a float32) Vec3 {
	return Vec3{v[0] * a, v[1] * a, v[2] * a}
}

func (v Vec3) Sub(a Vec3) Vec3 {
	return Vec3{v[0] - a[0], v[1] - a[1], v[2] - a[2]}
}

func (v Vec3) Add(a Vec3) Vec3 {
	return Vec3{v[0] + a[0], v[1] + a[1], v[2] + a[2]}
}

func (v Vec3) Dot(a Vec3) float32 {
	return v[0]*a[0] + v[1]*a[1] + v[2]*a[2]
}

func (v Vec3) CrossNormSq(a Vec3) float32 {
	d := v.Dot(a)
	return v.NormSq()*a.NormSq() - d*d
}

func (v Vec3) Cross(a Vec3) Vec3 {
	return Vec3{
		v[1]*a[2] - v[2]*a[1],
		v[2]*a[0] - v[0]*a[2],
		v[0]*a[1] - v[1]*a[0],
	}
}
