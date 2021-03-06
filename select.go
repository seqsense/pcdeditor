package main

import (
	"math"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

const (
	selectBitmaskCropped         = 0x00000001
	selectBitmaskSelected        = 0x00000002
	selectBitmaskNearCursor      = 0x00000004
	selectBitmaskOnScreen        = 0x00000008
	selectBitmaskExclude         = 0x80000000
	selectBitmaskSegmentSelected = 0x00000010
)

const (
	pointSelectRange = 0.1
	rectSelectRange  = 0.2
)

func selectPointOrtho(modelViewMatrix, projectionMatrix *mat.Mat4, x, y, width, height int, depth *mat.Vec3) *mat.Vec3 {
	a := projectionMatrix.Mul(*modelViewMatrix)

	var d float32
	if depth != nil {
		dp := a.Transform(*depth)
		d = dp[2]
	}

	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), d)

	target := a.Inv().Transform(pos)
	return &target
}

func screenPosVec(x, y, width, height int, projectionMatrix, modelViewMatrix *mat.Mat4) (*mat.Vec3, *mat.Mat4) {
	pos := mat.NewVec3(
		float32(x)*2/float32(width)-1,
		1-float32(y)*2/float32(height), -1)

	a := projectionMatrix.Mul(*modelViewMatrix).Inv()

	return &pos, &a
}

func perspectiveOriginDirFromPosVec(pos *mat.Vec3, a *mat.Mat4, modelViewMatrix *mat.Mat4) (*mat.Vec3, *mat.Vec3) {
	target := a.Transform(*pos)
	origin := modelViewMatrix.InvAffine().TransformAffine(mat.NewVec3(0, 0, 0))
	dir := target.Sub(origin).Normalized()

	return &origin, &dir
}

func perspectiveOriginDir(x, y, width, height int, projectionMatrix, modelViewMatrix *mat.Mat4) (*mat.Vec3, *mat.Vec3) {
	pos, a := screenPosVec(x, y, width, height, projectionMatrix, modelViewMatrix)
	return perspectiveOriginDirFromPosVec(pos, a, modelViewMatrix)
}

func selectPoint(pp *pc.PointCloud, selMask []uint32, projectionType ProjectionType, modelViewMatrix, projectionMatrix *mat.Mat4, x, y, width, height int, rangeMax float32) (*mat.Vec3, bool) {
	pos, a := screenPosVec(x, y, width, height, projectionMatrix, modelViewMatrix)

	it, err := pp.Vec3Iterator()
	if err != nil {
		return nil, false
	}

	var selected *mat.Vec3

	rangeMaxSq := rangeMax * rangeMax

	switch projectionType {
	case ProjectionPerspective:
		origin, dir := perspectiveOriginDirFromPosVec(pos, a, modelViewMatrix)
		vMin := float32(1000 * 1000)
		if selMask != nil {
			n := pp.Points
			for i := 0; i < n; i++ {
				if selMask[i]&(selectBitmaskCropped|selectBitmaskNearCursor|selectBitmaskOnScreen) != selectBitmaskNearCursor|selectBitmaskOnScreen {
					continue
				}
				p := it.Vec3At(i)
				pRel := origin.Sub(p)
				dot, distSq := pRel.Dot(*dir), pRel.NormSq()
				v := (distSq - dot*dot) + distSq/10000
				if v < vMin {
					vMin, selected = v, &p
				}
			}
		} else {
			// Full search for select box drag check
			for ; it.IsValid(); it.Incr() {
				p := it.Vec3()
				pRel := origin.Sub(p)
				dot := pRel.Dot(*dir)
				if dot < 0 {
					distSq := pRel.NormSq()
					dSq := distSq - dot*dot
					v := dSq + distSq/10000
					if v < vMin && dSq < rangeMaxSq && distSq > 1.0 {
						vMin, selected = v, &p
					}
				}
			}
		}
	case ProjectionOrthographic:
		o1 := a.TransformAffine(mat.NewVec3(pos[0], pos[1], 0))
		o2 := a.TransformAffine(mat.NewVec3(pos[0], pos[1], 1))
		oDiff := o2.Sub(o1)
		oDiffNormSq := oDiff.NormSq()

		dSqMin := rangeMaxSq
		for i := 0; it.IsValid(); func() {
			it.Incr()
			i++
		}() {
			if selMask != nil {
				if selMask[i]&selectBitmaskCropped != 0 {
					continue
				}
			}
			p := it.Vec3()
			dSq := oDiff.CrossNormSq(p.Sub(o1)) / oDiffNormSq
			if dSq < dSqMin {
				dSqMin = dSq
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

	z := m.TransformAffineZ(p3)
	v3 := v2n.Mul(z)

	return [8]mat.Vec3{
		pp[0], pp[1], pp[2], pp[3],
		pp[0].Add(v3), pp[1].Add(v3), pp[2].Add(v3), pp[3].Add(v3),
	}
}

func boxFromRect(min, max mat.Vec3) [8]mat.Vec3 {
	return [8]mat.Vec3{
		min,
		{min[0], max[1], min[2]},
		{max[0], max[1], min[2]},
		{max[0], min[1], min[2]},
		{min[0], min[1], max[2]},
		{min[0], max[1], max[2]},
		max,
		{max[0], min[1], max[2]},
	}
}

func dragTranslation(s, e mat.Vec3) mat.Mat4 {
	diff := e.Sub(s)
	return mat.Translate(diff[0], diff[1], diff[2])
}

func dragRotation(s, e mat.Vec3, rect []mat.Vec3, modelView *mat.Mat4) mat.Mat4 {
	if len(rect) <= 1 {
		return mat.Translate(0, 0, 0)
	}
	var center mat.Vec3
	for _, p := range rect {
		center = center.Add(p)
	}
	center = center.Mul(1 / float32(len(rect)))

	// Transform to view coordinate
	vCenter := modelView.Transform(center)
	vS := modelView.Transform(s)
	vE := modelView.Transform(e)
	vS[2], vE[2] = vCenter[2], vCenter[2]

	// Get view direction
	viewInv := modelView.Inv()
	camera := viewInv.Transform(mat.Vec3{})
	cameraFront := viewInv.Transform(mat.Vec3{0, 0, 1})
	dir := cameraFront.Sub(camera)

	// Calculate angle of dragged point around the center of the rect
	vSRel := vS.Sub(vCenter).Normalized()
	vERel := vE.Sub(vCenter).Normalized()
	ang0 := float32(math.Atan2(float64(vSRel[1]), float64(vSRel[0])))
	ang1 := float32(math.Atan2(float64(vERel[1]), float64(vERel[0])))
	return mat.Translate(center[0], center[1], center[2]).
		Mul(mat.Rotate(dir[0], dir[1], dir[2], ang1-ang0)).
		Mul(mat.Translate(-center[0], -center[1], -center[2]))
}

func cursorsToTrans(curs []mat.Vec3) mat.Mat4 {
	o := curs[0]
	x := curs[1].Sub(o)
	y := curs[2].Sub(o)
	z := curs[3].Sub(o)
	return mat.Mat4{
		x[0], x[1], x[2], 0,
		y[0], y[1], y[2], 0,
		z[0], z[1], z[2], 0,
		o[0], o[1], o[2], 1,
	}
}
