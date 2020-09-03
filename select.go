package main

import (
	"math"

	"github.com/seqsense/pcdviewer/mat"
	"github.com/seqsense/pcdviewer/pcd"
)

func selectPoint(pc *pcd.PointCloud, modelViewMatrix, projectionMatrix mat.Mat4, fov float64, x, y, width, height int) (*mat.Vec3, bool) {
	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), -1)

	a := projectionMatrix.Mul(modelViewMatrix).InvAffine()
	origin := a.Transform(mat.NewVec3(0, 0, -1-1.0/float32(math.Tan(fov))))
	target := a.Transform(pos)

	dir := target.Sub(origin).Normalized()

	it, err := pc.Float32Iterators("x", "y", "z")
	if err != nil {
		return nil, false
	}
	xi, yi, zi := it[0], it[1], it[2]
	var selected *mat.Vec3
	dSqMin := float32(0.1 * 0.1)
	vMin := float32(1000 * 1000)
	for xi.IsValid() {
		p := mat.NewVec3(xi.Float32(), yi.Float32(), zi.Float32())
		pRel := origin.Sub(p)
		dot := pRel.Dot(dir)
		if dot < 0 {
			distSq := pRel.NormSq()
			dSq := distSq - dot*dot
			v := distSq/10000 + dSq
			if dSq < dSqMin && v < vMin {
				vMin = v
				selected = &p
			}
		}
		xi.Incr()
		yi.Incr()
		zi.Incr()
	}
	if selected != nil {
		return selected, true
	}
	return nil, false
}
