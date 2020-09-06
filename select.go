package main

import (
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

func selectPoint(pc *pcd.PointCloud, modelViewMatrix, projectionMatrix mat.Mat4, x, y, width, height int) (*mat.Vec3, bool) {
	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), -1)

	a := projectionMatrix.Mul(modelViewMatrix).Inv()
	origin := modelViewMatrix.InvAffine().Transform((mat.NewVec3(0, 0, 0)))
	target := a.Transform(pos)

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

func rectFrom3(p0, p1, p2 mat.Vec3) (mat.Vec3, mat.Vec3) {
	base := p1.Sub(p0)
	proj := p0.Add(
		base.Mul(base.Dot(p2.Sub(p0)) / base.NormSq()))
	perp := p2.Sub(proj)
	return p1.Add(perp), p0.Add(perp)
}
