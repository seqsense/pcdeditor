package main

import (
	"errors"

	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
	"github.com/seqsense/pcdeditor/pcd/filter/voxelgrid"
)

const (
	defaultResolution             = 0.05
	defaultSelectRangePerspective = 0.05
	defaultSelectRangeOrtho       = 10.0
	defaultMapAlpha               = 0.3
	defaultZMin                   = -5.0
	defaultZMax                   = 5.0
)

type pcdIO interface {
	readPCD(path string) (*pcd.PointCloud, error)
	writePCD(path string, pc *pcd.PointCloud) error
	exportPCD(pc *pcd.PointCloud) (interface{}, error)
}

type mapIO interface {
	readMap(yamlPath, imgPath string) (*occupancyGrid, mapImage, error)
}

type commandContext struct {
	*editor
	pcdIO             pcdIO
	mapIO             mapIO
	pointCloudUpdated bool

	selectRange            *float32
	selectRangeOrtho       float32
	selectRangePerspective float32

	selected []mat.Vec3

	rectUpdated bool
	rect        []mat.Vec3
	rectCenter  []mat.Vec3

	mapInfo *occupancyGrid
	mapImg  mapImage

	mapAlpha float32

	zMin, zMax     float32
	projectionType ProjectionType
}

func newCommandContext(pcdio pcdIO, mapio mapIO) *commandContext {
	c := &commandContext{
		selectRangeOrtho:       defaultSelectRangeOrtho,
		selectRangePerspective: defaultSelectRangePerspective,
		editor:                 &editor{},
		pcdIO:                  pcdio,
		mapIO:                  mapio,
		mapAlpha:               defaultMapAlpha,
		zMin:                   defaultZMin,
		zMax:                   defaultZMax,
		projectionType:         ProjectionPerspective,
	}
	c.selectRange = &c.selectRangePerspective
	return c
}

func (c *commandContext) Map() (*occupancyGrid, mapImage, bool) {
	return c.mapInfo, c.mapImg, c.mapInfo != nil
}

func (c *commandContext) MapAlpha() float32 {
	return c.mapAlpha
}

func (c *commandContext) SetMapAlpha(a float32) {
	c.mapAlpha = a
}

func (c *commandContext) ZRange() (float32, float32) {
	return c.zMin, c.zMax
}

func (c *commandContext) SetZRange(zMin, zMax float32) {
	c.zMin, c.zMax = zMin, zMax
}

func (c *commandContext) PointCloud() (*pcd.PointCloud, bool, bool) {
	updated := c.pointCloudUpdated
	c.pointCloudUpdated = false
	return c.editor.pc, updated, c.editor.pc != nil
}

func (c *commandContext) PointCloudCropped() (*pcd.PointCloud, bool, bool) {
	updated := c.pointCloudUpdated
	c.pointCloudUpdated = false
	return c.editor.pcCrop, updated, c.editor.pcCrop != nil
}

func (c *commandContext) Crop() bool {
	m, ok := c.SelectMatrix()
	if !ok {
		c.editor.Crop(mat.Mat4{})
		c.pointCloudUpdated = true
		return false
	}
	c.editor.Crop(m)
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) updateRect() {
	switch len(c.selected) {
	case 0, 1, 2:
		c.rect = c.selected
	case 3:
		pp := rectFrom3(c.selected[0], c.selected[1], c.selected[2])
		c.rectCenter = []mat.Vec3{pp[0], pp[1], pp[2], pp[3]}
		norm := (pp[1].Sub(pp[0])).Cross(pp[3].Sub(pp[0])).Normalized().Mul(*c.selectRange)
		c.rect = []mat.Vec3{
			pp[0].Add(norm), pp[1].Add(norm), pp[2].Add(norm), pp[3].Add(norm),
			pp[0].Sub(norm), pp[1].Sub(norm), pp[2].Sub(norm), pp[3].Sub(norm),
		}
	case 4:
		pp := boxFrom4(c.selected[0], c.selected[1], c.selected[2], c.selected[3])
		c.rectCenter = []mat.Vec3{pp[0], pp[1], pp[2], pp[3]}
		c.rect = pp[:]
	}
	switch len(c.selected) {
	case 3, 4:
		c.rect = append(c.rect,
			c.rect[0], c.rect[2], c.rect[6], c.rect[4],
			c.rect[1], c.rect[3], c.rect[7], c.rect[5],
		)
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
	*c.selectRange = r
	c.updateRect()
}

func (c *commandContext) SelectRange() float32 {
	return *c.selectRange
}

func (c *commandContext) SetCursor(i int, p mat.Vec3) bool {
	if i < len(c.selected) {
		c.selected[i] = p
		c.updateRect()
		return true
	}
	if i == len(c.selected) && i < 5 {
		c.selected = append(c.selected, p)
		c.updateRect()
		return true
	}
	return false
}

func (c *commandContext) ProjectionType() ProjectionType {
	return c.projectionType
}

func (c *commandContext) SetProjectionType(p ProjectionType) {
	c.projectionType = p
	switch p {
	case ProjectionOrthographic:
		c.selectRange = &c.selectRangeOrtho
	case ProjectionPerspective:
		c.selectRange = &c.selectRangePerspective
	default:
		panic("unknown projection type")
	}
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

func (c *commandContext) SelectMatrix() (mat.Mat4, bool) {
	switch len(c.selected) {
	case 3:
		v0, v1 := c.rectCenter[1].Sub(c.rectCenter[0]), c.rectCenter[3].Sub(c.rectCenter[0])
		v0n, v1n := v0.Normalized(), v1.Normalized()
		v2 := v0n.Cross(v1n).Mul(*c.selectRange * 2)
		origin := c.rectCenter[0].Sub(v2.Mul(0.5))
		m := (mat.Mat4{
			v0[0], v0[1], v0[2], 0,
			v1[0], v1[1], v1[2], 0,
			v2[0], v2[1], v2[2], 0,
			0, 0, 0, 1,
		}).InvAffine().
			MulAffine(mat.Translate(-origin[0], -origin[1], -origin[2]))

		return m, true
	case 4:
		v0, v1 := c.rect[1].Sub(c.rect[0]), c.rect[3].Sub(c.rect[0])
		v2 := c.rect[4].Sub(c.rect[0])
		m := (mat.Mat4{
			v0[0], v0[1], v0[2], 0,
			v1[0], v1[1], v1[2], 0,
			v2[0], v2[1], v2[2], 0,
			0, 0, 0, 1,
		}).InvAffine().
			MulAffine(mat.Translate(-c.rect[0][0], -c.rect[0][1], -c.rect[0][2]))

		return m, true
	default:
		return mat.Mat4{}, false
	}
}

func (c *commandContext) filter() (func(p mat.Vec3) bool, bool) {
	m, ok := c.SelectMatrix()
	if !ok {
		return nil, false
	}

	return mat4Filter(m).Filter, true
}

func (c *commandContext) AddSurface(resolution float32) bool {
	if len(c.selected) != 3 {
		return false
	}
	if resolution <= 0.0 {
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
	c.editor.merge(pcNew)
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) Delete() bool {
	filter, ok := c.filter()
	if !ok {
		return false
	}
	cropMat := c.editor.cropMatrix
	if cropMat[15] != 0.0 {
		selectFilter := filter
		cropFilter := mat4Filter(cropMat).Filter
		filter = func(p mat.Vec3) bool {
			if selectFilter(p) {
				return true
			}
			return cropFilter(p)
		}
	}
	c.editor.passThrough(filter)
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) VoxelFilter(resolution float32) error {
	m, selected := c.SelectMatrix()
	var pc *pcd.PointCloud
	if selected {
		var err error
		if pc, err = passThrough(c.editor.pc, mat4Filter(m).FilterInv); err != nil {
			return err
		}
	} else {
		pc = c.editor.pc
	}

	vg := voxelgrid.New(mat.Vec3{resolution, resolution, resolution})
	pcFiltered, err := vg.Filter(pc)
	if err != nil {
		return err
	}

	if selected {
		filter, ok := c.filter()
		if !ok {
			return errors.New("failed to create filter")
		}
		c.editor.passThrough(filter)
		c.editor.pop()
		c.editor.merge(pcFiltered)
	} else {
		if err := c.editor.SetPointCloud(pcFiltered); err != nil {
			return err
		}
	}

	c.pointCloudUpdated = true
	return nil
}

func (c *commandContext) Label(l uint32) bool {
	filter, ok := c.filter()
	if !ok {
		return false
	}
	c.editor.label(func(p mat.Vec3) (uint32, bool) {
		if filter(p) {
			return 0, false
		}
		return l, true
	})
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) Undo() bool {
	c.pointCloudUpdated = true
	return c.editor.Undo()
}

func (c *commandContext) LoadPCD(path string) error {
	p, err := c.pcdIO.readPCD(path)
	if err != nil {
		return err
	}
	if err := c.editor.SetPointCloud(p); err != nil {
		return err
	}

	c.pointCloudUpdated = true
	return nil
}

func (c *commandContext) Load2D(yamlPath, imgPath string) error {
	mi, img, err := c.mapIO.readMap(yamlPath, imgPath)
	if err != nil {
		c.mapInfo = nil
		return err
	}
	c.mapInfo = mi
	c.mapImg = img

	c.pointCloudUpdated = true
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

func (c *commandContext) ExportPCD() (interface{}, error) {
	if c.editor.pc == nil {
		return nil, errors.New("no pointcloud")
	}
	blob, err := c.pcdIO.exportPCD(c.editor.pc)
	if err != nil {
		return nil, err
	}
	return blob, nil
}