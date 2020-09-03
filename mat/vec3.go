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

func (m Mat4) TransformAffine(a Vec3) Vec3 {
	var out Vec3
	out[0] = m[4*0+0]*a[0] + m[4*1+0]*a[1] + m[4*2+0]*a[2] + m[4*3+0]
	out[1] = m[4*0+1]*a[0] + m[4*1+1]*a[1] + m[4*2+1]*a[2] + m[4*3+1]
	out[2] = m[4*0+2]*a[0] + m[4*1+2]*a[1] + m[4*2+2]*a[2] + m[4*3+2]
	return out
}

func (m Mat4) Transform(a Vec3) Vec3 {
	var out Vec3
	out[0] = m[4*0+0]*a[0] + m[4*1+0]*a[1] + m[4*2+0]*a[2] + m[4*3+0]
	out[1] = m[4*0+1]*a[0] + m[4*1+1]*a[1] + m[4*2+1]*a[2] + m[4*3+1]
	out[2] = m[4*0+2]*a[0] + m[4*1+2]*a[1] + m[4*2+2]*a[2] + m[4*3+2]
	return out
}
