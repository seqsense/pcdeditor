package mat

type Vec3 [4]float32

func NewVec3(x, y, z float32) Vec3 {
	return Vec3{x, y, z, 1}
}

func (m Mat4) TransformAffine(a Vec3) Vec3 {
	var out Vec3
	out[0] = m[4*0+0]*a[0] + m[4*1+0]*a[1] + m[4*2+0]*a[2] + m[4*3+0]
	out[1] = m[4*0+1]*a[0] + m[4*1+1]*a[1] + m[4*2+1]*a[2] + m[4*3+1]
	out[2] = m[4*0+2]*a[0] + m[4*1+2]*a[1] + m[4*2+2]*a[2] + m[4*3+2]
	out[3] = 1
	return out
}

func (m Mat4) Transform(a Vec3) Vec3 {
	var out Vec3
	out[0] = m[4*0+0]*a[0] + m[4*1+0]*a[1] + m[4*2+0]*a[2] + m[4*3+0]
	out[1] = m[4*0+1]*a[0] + m[4*1+1]*a[1] + m[4*2+1]*a[2] + m[4*3+1]
	out[2] = m[4*0+2]*a[0] + m[4*1+2]*a[1] + m[4*2+2]*a[2] + m[4*3+2]
	out[3] = m[4*0+3]*a[0] + m[4*1+3]*a[1] + m[4*2+3]*a[2] + m[4*3+3]
	return out
}
