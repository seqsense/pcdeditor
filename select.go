package main

import (
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

func selectPointOrtho(modelViewMatrix, projectionMatrix mat.Mat4, x, y, width, height int) *mat.Vec3 {
	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), 0)

	a := projectionMatrix.Mul(modelViewMatrix).Inv()
	target := a.Transform(pos)
	return &target
}

func selectPoint(pc *pcd.PointCloud, projectionType ProjectionType, modelViewMatrix, projectionMatrix mat.Mat4, x, y, width, height int) (*mat.Vec3, bool) {
	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), -1)

	a := projectionMatrix.Mul(modelViewMatrix).Inv()
	target := a.Transform(pos)

	var origin mat.Vec3

	switch projectionType {
	case ProjectionPerspective:
		origin = modelViewMatrix.InvAffine().Transform(mat.NewVec3(0, 0, 0))
	case ProjectionOrthographic:
		origin = a.Transform(mat.NewVec3(pos[0], pos[1], 1))
	}
	dir := target.Sub(origin).Normalized()

	it, err := pc.Vec3Iterator()
	if err != nil {
		return nil, false
	}
	var selected *mat.Vec3
	dSqMin := float32(0.1 * 0.1)
	vMin := float32(1000 * 1000)
	for ; it.IsValid(); it.Incr() {
		p := it.Vec3()
		pRel := origin.Sub(p)
		dot := pRel.Dot(dir)
		if dot < 0 {
			distSq := pRel.NormSq()
			dSq := distSq - dot*dot
			v := distSq/10000 + dSq
			if dSq < dSqMin && v < vMin && distSq > 1 {
				vMin = v
				selected = &p
			}
		}
	}
	if selected != nil {
		return selected, true
	}
	return nil, false
}

func rectFrom3(p0, p1, p2 mat.Vec3) [4]mat.Vec3 {
	base := p1.Sub(p0)
	proj := p0.Add(
		base.Mul(base.Dot(p2.Sub(p0)) / base.NormSq()))
	perp := p2.Sub(proj)
	return [4]mat.Vec3{p0, p1, p1.Add(perp), p0.Add(perp)}
}

func boxFrom4(p0, p1, p2, p3 mat.Vec3) [8]mat.Vec3 {
	pp := rectFrom3(p0, p1, p2)
	v0n, v1n := pp[1].Sub(p0).Normalized(), pp[3].Sub(p0).Normalized()
	v2n := v0n.Cross(v1n)
	m := (mat.Mat4{
		v0n[0], v0n[1], v0n[2], 0,
		v1n[0], v1n[1], v1n[2], 0,
		v2n[0], v2n[1], v2n[2], 0,
		0, 0, 0, 1,
	}).InvAffine().MulAffine(mat.Translate(-p0[0], -p0[1], -p0[2]))

	z := m.TransformZ(p3)
	v3 := v2n.Mul(z)

	return [8]mat.Vec3{
		pp[0], pp[1], pp[2], pp[3],
		pp[0].Add(v3), pp[1].Add(v3), pp[2].Add(v3), pp[3].Add(v3),
	}
}
