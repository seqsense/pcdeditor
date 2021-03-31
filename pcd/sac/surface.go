package sac

import (
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
	"github.com/seqsense/pcdeditor/pcd/storage/voxelgrid"
)

type voxelGridSurfaceModel struct {
	vg                   *voxelgrid.VoxelGrid
	ra                   pcd.Vec3RandomAccessor
	vgMin, vgMax, vgSize mat.Vec3
}

const (
	sqrt3   = 1.732050808
	epsilon = 0.01
)

func NewVoxelGridSurfaceModel(vg *voxelgrid.VoxelGrid, ra pcd.Vec3RandomAccessor) *voxelGridSurfaceModel {
	vgMin, vgMax := vg.MinMax()
	vgSize := vgMax.Sub(vgMin)
	return &voxelGridSurfaceModel{
		vg:     vg,
		ra:     ra,
		vgMin:  vgMin,
		vgMax:  vgMax,
		vgSize: vgSize,
	}
}

func (voxelGridSurfaceModel) NumRange() (min, max int) {
	return 3, 3
}

func (m *voxelGridSurfaceModel) Fit(ids []int) (ModelCoefficients, bool) {
	if len(ids) != 3 {
		return nil, false
	}

	p0, p1, p2 := m.ra.Vec3At(ids[0]).Sub(m.vgMin), m.ra.Vec3At(ids[1]).Sub(m.vgMin), m.ra.Vec3At(ids[2]).Sub(m.vgMin)
	v1, v2 := p1.Sub(p0), p2.Sub(p0)

	// Calculate normal vector of the surface made by given three points.
	norm := v1.Cross(v2)
	if nearZeroSq(norm.NormSq()) {
		return nil, false
	}

	// Plane equation: norm[0]*x + norm[1]*y + norm[2]*z = d
	norm = norm.Normalized()
	d := norm.Dot(p0)

	// Calculate edges on the voxelgrid boundary made by the plane.
	nValid := [3]bool{!nearZero(norm[0]), !nearZero(norm[1]), !nearZero(norm[2])}
	var o [3][4]mat.Vec3
	vgn := norm.ElementMul(m.vgSize)
	// List all crossing points of the plane and boundary box.
	if nValid[0] {
		o[0][0] = mat.Vec3{(d - vgn[1] - vgn[2]) / norm[0], m.vgSize[1], m.vgSize[2]} // y+z+
		o[0][1] = mat.Vec3{(d - vgn[1]) / norm[0], m.vgSize[1], 0}                    // y+z-
		o[0][2] = mat.Vec3{(d - vgn[2]) / norm[0], 0, m.vgSize[2]}                    // y-z+
		o[0][3] = mat.Vec3{d / norm[0], 0, 0}                                         // y-z-
	}
	if nValid[1] {
		o[1][0] = mat.Vec3{m.vgSize[0], (d - vgn[0] - vgn[2]) / norm[1], m.vgSize[2]} // x+z+
		o[1][1] = mat.Vec3{m.vgSize[0], (d - vgn[0]) / norm[1], 0}                    // x+z-
		o[1][2] = mat.Vec3{0, (d - vgn[2]) / norm[1], m.vgSize[2]}                    // x-z+
		o[1][3] = mat.Vec3{0, d / norm[1], 0}                                         // x-z-
	}
	if nValid[2] {
		o[2][0] = mat.Vec3{m.vgSize[0], m.vgSize[1], (d - vgn[0] - vgn[1]) / norm[2]} // x+y+
		o[2][1] = mat.Vec3{m.vgSize[0], 0, (d - vgn[0]) / norm[2]}                    // x+y-
		o[2][2] = mat.Vec3{0, m.vgSize[1], (d - vgn[1]) / norm[2]}                    // x-y+
		o[2][3] = mat.Vec3{0, 0, d / norm[2]}                                         // x-y-
	}

	type point struct {
		a, i int
	}
	var edge [3][4][]point
	isInside := func(p mat.Vec3) bool {
		return !(p[0] < 0 || m.vgSize[0] < p[0] || p[1] < 0 || m.vgSize[1] < p[1] || p[2] < 0 || m.vgSize[2] < p[2])
	}
	// List all edges of the plane cut by boundary box.
	for _, l := range []struct {
		a0, i0, a1, i1 int
	}{
		// Commented elements are the duplication of A->B and B->A.
		{0, 0, 1, 0}, {0, 0, 1, 2}, {0, 0, 2, 0}, {0, 0, 2, 2}, // (y+z+)<=>(x+z+), (y+z+)<=>(x-z+), (y+z+)<=>(x+y+), (y+z+)<=>(x-y+)
		{0, 1, 1, 1}, {0, 1, 1, 3}, {0, 1, 2, 0}, {0, 1, 2, 2}, // (y+z-)<=>(x+z-), (y+z-)<=>(x-z-), (y+z-)<=>(x+y+), (y+z-)<=>(x-y+)
		{0, 2, 1, 0}, {0, 2, 1, 2}, {0, 2, 2, 1}, {0, 2, 2, 3}, // (y-z+)<=>(x+z+), (y-z+)<=>(x-z+), (y-z+)<=>(x+y-), (y-z+)<=>(x-y-)
		{0, 3, 1, 1}, {0, 3, 1, 3}, {0, 3, 2, 1}, {0, 3, 2, 3}, // (y-z-)<=>(x+z-), (y-z-)<=>(x-z-), (y-z-)<=>(x+y-), (y-z-)<=>(x-y-)

		/*{1, 0, 0, 0}, {1, 0, 0, 2},*/ {1, 0, 2, 0}, {1, 0, 2, 1}, // (x+z+)<=>(y+z+), (x+z+)<=>(y-z+), (x+z+)<=>(x+y+), (x+z+)<=>(x+y-)
		/*{1, 1, 0, 1}, {1, 1, 0, 3},*/ {1, 1, 2, 0}, {1, 1, 2, 1}, // (x+z-)<=>(y+z-), (x+z-)<=>(y-z-), (x+z-)<=>(x+y+), (x+z-)<=>(x+y-)
		/*{1, 2, 0, 0}, {1, 2, 0, 2},*/ {1, 2, 2, 2}, {1, 2, 2, 3}, // (x-z+)<=>(y+z+), (x-z+)<=>(y-z+), (x-z+)<=>(x-y+), (x-z+)<=>(x-y-)
		/*{1, 3, 0, 1}, {1, 3, 0, 3},*/ {1, 3, 2, 2}, {1, 3, 2, 3}, // (x-z-)<=>(y+z-), (x-z-)<=>(y-z-), (x-z-)<=>(x-y+), (x-z-)<=>(x-y-)

		/*{2, 0, 0, 0}, {2, 0, 0, 1}, {2, 0, 1, 0}, {2, 0, 1, 1},*/ // (x+y+)<=>(y+z+), (x+y+)<=>(y+z-), (x+y+)<=>(x+z+), (x+y+)<=>(x+z-)
		/*{2, 1, 0, 2}, {2, 1, 0, 3}, {2, 1, 1, 0}, {2, 1, 1, 1},*/ // (x+y-)<=>(y-z+), (x+y-)<=>(y-z-), (x+y-)<=>(x+z+), (x+y-)<=>(x+z-)
		/*{2, 2, 0, 0}, {2, 2, 0, 1}, {2, 2, 1, 2}, {2, 2, 1, 3},*/ // (x-y+)<=>(y+z+), (x-y+)<=>(y+z-), (x-y+)<=>(x-z+), (x-y+)<=>(x-z-)
		/*{2, 3, 0, 2}, {2, 3, 0, 3}, {2, 3, 1, 2}, {2, 3, 1, 3},*/ // (x-y-)<=>(y-z+), (x-y-)<=>(y-z-), (x-y-)<=>(x-z+), (x-y-)<=>(x-z-)

		{0, 0, 0, 2}, {0, 0, 0, 1}, {0, 1, 0, 3}, {0, 3, 0, 2}, // (y+z+)<=>(y-z+), (y+z+)<=>(y+z-), (y+z-)<=>(y-z-), (y-z-)<=>(y-z+)
		{1, 0, 1, 2}, {1, 0, 1, 1}, {1, 1, 1, 3}, {1, 3, 1, 2}, // (x+z+)<=>(x-z+), (x+z+)<=>(x+z-), (x+z-)<=>(x-z-), (x-z-)<=>(x-z+)
		{2, 0, 2, 2}, {2, 0, 2, 1}, {2, 1, 2, 3}, {2, 3, 2, 2}, // (x+y+)<=>(x-y+), (x+y+)<=>(x+y-), (x+y-)<=>(x-y-), (x-y-)<=>(x-y+)
	} {
		if !nValid[l.a0] || !nValid[l.a1] {
			continue
		}
		if isInside(o[l.a0][l.i0]) && isInside(o[l.a1][l.i1]) && !nearZeroSq(o[l.a0][l.i0].Sub(o[l.a1][l.i1]).NormSq()) {
			edge[l.a0][l.i0] = append(edge[l.a0][l.i0], point{l.a1, l.i1})
			edge[l.a1][l.i1] = append(edge[l.a1][l.i1], point{l.a0, l.i0})
		}
	}

	// Remove duplication of the edges.
	for a := 0; a < 3; a++ {
		for i := 0; i < 4; i++ {
			es := edge[a][i]
			es2 := make([]point, 0, len(es))
			for j, e := range es {
				ok := true
				for k := j + 1; k < len(es); k++ {
					if nearZeroSq(o[e.a][e.i].Sub(o[es[k].a][es[k].i]).NormSq()) {
						ok = false
						break
					}
				}
				if ok {
					es2 = append(es2, e)
				}
			}
			edge[a][i] = es2
		}
	}

	// Find largest two connected edges of the cut section of the boundary box.
	// Voxels on the surface can be approximately scanned by weighted average of the two edges.
	var aO, iO int
	var maxLenSq float32
	for a := 0; a < 3; a++ {
		for i := 0; i < 4; i++ {
			es := edge[a][i]
			if len(es) != 2 {
				continue
			}
			var l float32
			for _, e := range es {
				l += o[a][i].Sub(o[e.a][e.i]).NormSq()
			}
			if l > maxLenSq {
				maxLenSq = l
				aO, iO = a, i
			}
		}
	}
	if maxLenSq == 0 {
		// Surface not found
		return nil, false
	}

	es := edge[aO][iO]
	o0, o1, o2 := o[es[0].a][es[0].i], o[aO][iO], o[es[1].a][es[1].i]
	ov1, ov2 := o0.Sub(o1), o2.Sub(o1)

	// Set a voxel scan interval to make it visited at least once per voxel.
	r := m.vg.Resolution() / sqrt3

	return &voxelGridSurfaceModelCoefficients{
		model:  m,
		origin: o1.Add(m.vgMin),
		v1:     ov1,
		v2:     ov2,
		l1:     r / ov1.Norm(),
		l2:     r / ov2.Norm(),
		norm:   norm,
		d:      d,
	}, true
}

func nearZero(a float32) bool {
	return -epsilon < a && a < epsilon
}

func nearZeroSq(a float32) bool {
	return a < epsilon*epsilon
}

type voxelGridSurfaceModelCoefficients struct {
	model *voxelGridSurfaceModel

	origin mat.Vec3
	v1, v2 mat.Vec3
	l1, l2 float32

	norm mat.Vec3
	d    float32
}

func (c *voxelGridSurfaceModelCoefficients) Evaluate() int {
	added := make([]bool, c.model.vg.Len())
	var cnt int

	for a := float32(0); a <= 1; a += c.l1 {
		for b := float32(0); b <= 1; b += c.l2 {
			p := c.origin.Add(c.v1.Mul(a)).Add(c.v2.Mul(b))
			addr, ok := c.model.vg.Addr(p)
			if !ok {
				continue
			}
			if !added[addr] {
				added[addr] = true
				cnt += len(c.model.vg.GetByAddr(addr))
			}
		}
	}
	return cnt
}

func (c *voxelGridSurfaceModelCoefficients) Inliers(d float32) []int {
	vgMin := c.model.vgMin

	n := c.model.ra.Len()
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		p := c.model.ra.Vec3At(i)
		dd := c.norm.Dot(p.Sub(vgMin)) - c.d
		if -d < dd && dd < d {
			out = append(out, i)
		}
	}
	return out
}

func (c *voxelGridSurfaceModelCoefficients) IsIn(p mat.Vec3, d float32) bool {
	dd := c.norm.Dot(p.Sub(c.model.vgMin)) - c.d
	return -d < dd && dd < d
}
