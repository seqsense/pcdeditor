package main

import (
	"errors"

	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

const (
	defaultResolution  = 0.05
	defaultSelectRange = 0.05
)

type pcdIO interface {
	readPCD(path string) (*pcd.PointCloud, error)
	writePCD(path string, pc *pcd.PointCloud) error
	exportPCD(filename string, pc *pcd.PointCloud) error
}

type commandContext struct {
	*editor
	pcdIO pcdIO

	selectRange float32

	selected []mat.Vec3

	rectUpdated bool
	rect        []mat.Vec3
	rectCenter  []mat.Vec3
}

func newCommandContext(pcdio pcdIO) *commandContext {
	return &commandContext{
		selectRange: defaultSelectRange,
		editor:      &editor{},
		pcdIO:       pcdio,
	}
}

func (c *commandContext) updateRect() {
	switch len(c.selected) {
	case 0, 1, 2:
		c.rect = c.selected
	case 3:
		p0, p1 := c.selected[0], c.selected[1]
		p2, p3 := rectFrom3(p0, p1, c.selected[2])
		c.rectCenter = []mat.Vec3{p0, p1, p2, p3}
		norm := (p1.Sub(p0)).Cross(p3.Sub(p0)).Normalized().Mul(c.selectRange)
		c.rect = []mat.Vec3{
			p0.Add(norm), p1.Add(norm), p2.Add(norm), p3.Add(norm),
			p0.Sub(norm), p1.Sub(norm), p2.Sub(norm), p3.Sub(norm),
		}
	}
	c.rectUpdated = true
}

func (c *commandContext) Rect() ([]mat.Vec3, bool) {
	updated := c.rectUpdated
	c.rectUpdated = false
	return c.rect, updated
}

func (c *commandContext) RectCenter() []mat.Vec3 {
	return c.rectCenter
}

func (c *commandContext) SetSelectRange(r float32) {
	if r < 0 {
		r = 0
	}
	c.selectRange = r
	c.updateRect()
}

func (c *commandContext) SelectRange() float32 {
	return c.selectRange
}

func (c *commandContext) SetCursor(i int, p mat.Vec3) bool {
	if i < len(c.selected) {
		c.selected[i] = p
		c.updateRect()
		return true
	}
	if i == len(c.selected) && i < 4 {
		c.selected = append(c.selected, p)
		c.updateRect()
		return true
	}
	return false
}

func (c *commandContext) Cursors() []mat.Vec3 {
	return c.selected
}

func (c *commandContext) UnsetCursors() {
	c.selected = nil
	c.updateRect()
}

func (c *commandContext) SnapVertical() {
	if len(c.selected) > 2 {
		c.selected[2][0] = c.selected[0][0]
		c.selected[2][1] = c.selected[0][1]
		c.updateRect()
	}
}

func (c *commandContext) SnapHorizontal() {
	if len(c.selected) > 1 {
		c.selected[1][2] = c.selected[0][2]
	}
	if len(c.selected) > 2 {
		c.selected[2][2] = c.selected[0][2]
	}
	c.updateRect()
}

func (c *commandContext) TransformCursors(m mat.Mat4) {
	for i := range c.selected {
		c.selected[i] = m.Transform(c.selected[i])
	}
	c.updateRect()
}

func (c *commandContext) filter() (func(p mat.Vec3) bool, bool) {
	if len(c.selected) != 3 {
		return nil, false
	}
	v0, v1 := c.rectCenter[1].Sub(c.rectCenter[0]), c.rectCenter[3].Sub(c.rectCenter[0])
	v0n, v1n := v0.Normalized(), v1.Normalized()
	v2n := v0n.Cross(v1n)
	m := (mat.Mat4{
		v0n[0], v0n[1], v0n[2], 0,
		v1n[0], v1n[1], v1n[2], 0,
		v2n[0], v2n[1], v2n[2], 0,
		0, 0, 0, 1,
	}).InvAffine().
		MulAffine(mat.Translate(-c.rectCenter[0][0], -c.rectCenter[0][1], -c.rectCenter[0][2]))
	l0 := v0.Norm()
	l1 := v1.Norm()

	return func(p mat.Vec3) bool {
		if z := m.TransformZ(p); z < -c.selectRange || c.selectRange < z {
			return true
		}
		if x := m.TransformX(p); x < 0 || l0 < x {
			return true
		}
		if y := m.TransformY(p); y < 0 || l1 < y {
			return true
		}
		return false
	}, true
}

func (c *commandContext) AddSurface(resolution float32) bool {
	if len(c.selected) != 3 {
		return false
	}
	v0, v1 := c.rectCenter[1].Sub(c.rectCenter[0]), c.rectCenter[3].Sub(c.rectCenter[0])
	v0n, v1n := v0.Normalized(), v1.Normalized()
	v2n := v0n.Cross(v1n)
	m := mat.Translate(c.rectCenter[0][0], c.rectCenter[0][1], c.rectCenter[0][2]).
		MulAffine(mat.Mat4{
			v0n[0], v0n[1], v0n[2], 0,
			v1n[0], v1n[1], v1n[2], 0,
			v2n[0], v2n[1], v2n[2], 0,
			0, 0, 0, 1,
		})
	l0 := v0.Norm()
	l1 := v1.Norm()

	w := int(l0 / resolution)
	h := int(l1 / resolution)
	pcNew := &pcd.PointCloud{
		PointCloudHeader: c.editor.pc.PointCloudHeader.Clone(),
		Points:           w * h,
		Data:             make([]byte, w*h*c.editor.pc.Stride()),
	}
	it, err := pcNew.Vec3Iterator()
	if err != nil {
		return false
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			it.SetVec3(
				m.Transform(mat.Vec3{float32(x) * resolution, float32(y) * resolution, 0}),
			)
			it.Incr()
		}
	}
	c.merge(pcNew)
	return true
}

func (c *commandContext) Delete() bool {
	filter, ok := c.filter()
	if !ok {
		return false
	}
	c.passThrough(filter)
	return true
}

func (c *commandContext) Label(l uint32) bool {
	filter, ok := c.filter()
	if !ok {
		return false
	}
	c.label(func(p mat.Vec3) (uint32, bool) {
		if filter(p) {
			return 0, false
		}
		return l, true
	})
	return true
}

func (c *commandContext) LoadPCD(path string) error {
	p, err := c.pcdIO.readPCD(path)
	if err != nil {
		return err
	}
	if err := c.SetPointCloud(p); err != nil {
		return err
	}
	return nil
}

func (c *commandContext) SavePCD(path string) error {
	if c.editor.pc == nil {
		return errors.New("no pointcloud")
	}
	if err := c.pcdIO.writePCD(path, c.editor.pc); err != nil {
		return err
	}
	return nil
}

func (c *commandContext) ExportPCD(path string) error {
	if c.editor.pc == nil {
		return errors.New("no pointcloud")
	}
	err := c.pcdIO.exportPCD(path, c.editor.pc)
	if err != nil {
		return err
	}
	return nil
}
