package main

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
	"github.com/seqsense/pcgol/pc/filter/voxelgrid"
	"github.com/seqsense/pcgol/pc/registration/icp"
	"github.com/seqsense/pcgol/pc/sac"
	vgs "github.com/seqsense/pcgol/pc/segmentation/voxelgrid"
	"github.com/seqsense/pcgol/pc/storage/kdtree"
)

const (
	defaultResolution             = 0.05
	defaultSelectRangePerspective = 0.05
	defaultSelectRangeOrtho       = 500.0
	defaultMapAlpha               = 0.3
	defaultZMin                   = -5.0
	defaultZMax                   = 5.0
	defaultPointSize              = 40.0
	defaultSegmentationDistance   = 0.08
	defaultSegmentationRange      = 5.0

	sacIterationCnt     = 20
	sacSurfacePointsMin = 50
)

type rangeType int

const (
	rangeTypeAuto = iota
	rangeTypePerspective
	rangeTypeOrtho
)

type pcdIO interface {
	importPCD(blob interface{}) (*pc.PointCloud, error)
	exportPCD(pp *pc.PointCloud) (interface{}, error)
}

type mapIO interface {
	readMap(yamlBlob, img interface{}) (*occupancyGrid, mapImage, error)
}

type selectMode int

const (
	selectModeRect selectMode = iota
	selectModeMask
	selectModeInsert
)

type commandContext struct {
	*editor
	pcdIO                pcdIO
	mapIO                mapIO
	pointCloudUpdated    bool
	subPointCloudUpdated bool
	mapUpdated           bool

	selectRange            *float32
	selectRangeOrtho       float32
	selectRangePerspective float32

	selected      []mat.Vec3
	selectedStack [][]mat.Vec3
	selectMask    []uint32

	rectUpdated bool
	rect        []mat.Vec3
	rectCenter  []mat.Vec3

	mapInfo *occupancyGrid
	mapImg  mapImage

	mapAlpha float32

	zMin, zMax     float32
	projectionType ProjectionType

	pointSize float32

	selectMode selectMode

	segmentationDistance, segmentationRange float32
}

func newCommandContext(pcdio pcdIO, mapio mapIO) *commandContext {
	c := &commandContext{
		selectRangeOrtho:       defaultSelectRangeOrtho,
		selectRangePerspective: defaultSelectRangePerspective,
		editor:                 newEditor(),
		pcdIO:                  pcdio,
		mapIO:                  mapio,
		mapAlpha:               defaultMapAlpha,
		zMin:                   defaultZMin,
		zMax:                   defaultZMax,
		projectionType:         ProjectionPerspective,
		pointSize:              defaultPointSize,
		segmentationDistance:   defaultSegmentationDistance,
		segmentationRange:      defaultSegmentationRange,
	}
	c.selectRange = &c.selectRangePerspective
	return c
}

func (c *commandContext) SelectMask() []uint32 {
	return c.selectMask
}

func (c *commandContext) SetSelectMask(mask []uint32) {
	c.selectMask = mask
}

func (c *commandContext) Map() (*occupancyGrid, mapImage, bool, bool) {
	updated := c.mapUpdated
	c.mapUpdated = false
	return c.mapInfo, c.mapImg, updated, c.mapInfo != nil
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

func (c *commandContext) PointSize() float32 {
	return c.pointSize
}

func (c *commandContext) SetPointSize(ps float32) error {
	if ps <= 0 {
		return errors.New("point size must be >0")
	}
	c.pointSize = ps
	return nil
}

func (c *commandContext) SegmentationParam() (float32, float32) {
	return c.segmentationDistance, c.segmentationRange
}

func (c *commandContext) SetSegmentationParam(dist, r float32) error {
	if dist <= 0 || r <= 0 {
		return errors.New("invalid segmentation param (D and R must be >0)")
	}
	if v := int(r / dist); v < 1 || 256 < v {
		return errors.New("invalid segmentation param (R/D must be 1-256)")
	}
	c.segmentationDistance, c.segmentationRange = dist, r
	return nil
}

func (c *commandContext) PointCloud() (*pc.PointCloud, bool, bool) {
	updated := c.pointCloudUpdated
	c.pointCloudUpdated = false
	return c.editor.pp, updated, c.editor.pp != nil
}

func (c *commandContext) SubPointCloud() (*pc.PointCloud, bool, bool) {
	updated := c.subPointCloudUpdated
	c.subPointCloudUpdated = false
	return c.editor.ppSub, updated, c.editor.ppSub != nil
}

func (c *commandContext) CropMatrix() mat.Mat4 {
	return c.editor.cropMatrix
}

func (c *commandContext) Crop() bool {
	m, ok := c.SelectMatrix()
	if !ok {
		c.editor.Crop(mat.Mat4{})
		return false
	}
	c.editor.Crop(m)
	return true
}

func (c *commandContext) updateRect() {
	if c.selectMode == selectModeInsert {
		b := boxFromRect(c.editor.ppSubRect.min, c.editor.ppSubRect.max)
		trans := cursorsToTrans(c.selected)
		for i := range b {
			b[i] = trans.Transform(b[i])
		}
		c.rect = b[:]
		c.rectUpdated = true
		return
	}
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

func (c *commandContext) RectCenterPos() mat.Vec3 {
	if len(c.rect) == 0 {
		return mat.Vec3{}
	}
	var center mat.Vec3
	for _, p := range c.rect {
		center = center.Add(p)
	}
	return center.Mul(1 / float32(len(c.rect)))
}

func (c *commandContext) SetSelectRange(t rangeType, r float32) {
	if r < 0 {
		r = 0
	}
	switch t {
	case rangeTypeAuto:
		*c.selectRange = r
	case rangeTypePerspective:
		c.selectRangePerspective = r
	case rangeTypeOrtho:
		c.selectRangeOrtho = r
	default:
		panic("invalid rangeType")
	}
	c.updateRect()
}

func (c *commandContext) SelectRange(t rangeType) float32 {
	switch t {
	case rangeTypeAuto:
		return *c.selectRange
	case rangeTypePerspective:
		return c.selectRangePerspective
	case rangeTypeOrtho:
		return c.selectRangeOrtho
	}
	panic("invalid rangeType")
}

func (c *commandContext) SelectMode() selectMode {
	return c.selectMode
}

func (c *commandContext) SetCursor(i int, p mat.Vec3) bool {
	if c.selectMode == selectModeInsert {
		return false
	}
	c.selectMode = selectModeRect
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
	if c.selectMode == selectModeInsert {
		// Clear sub cloud to leave insert mode
		_ = c.editor.SetPointCloud(nil, cloudSub)
	}
	c.selectMode = selectModeRect
	c.selected = nil
	c.updateRect()
}

func (c *commandContext) PushCursors() {
	if len(c.selected) == 0 {
		return
	}
	var copied []mat.Vec3
	for _, s := range c.selected {
		copied = append(copied, mat.Vec3{s[0], s[1], s[2]})
	}
	c.selectedStack = append(c.selectedStack, copied)
	c.updateRect()
}

func (c *commandContext) PopCursors() {
	if len(c.selected) == 0 {
		return
	}
	c.selected = c.selectedStack[len(c.selectedStack)-1]
	c.selectedStack = c.selectedStack[:len(c.selectedStack)-1]
	c.updateRect()
}

func (c *commandContext) SnapVertical() {
	if c.selectMode == selectModeInsert {
		return
	}
	if len(c.selected) > 2 {
		c.selected[2][0] = c.selected[0][0]
		c.selected[2][1] = c.selected[0][1]
		c.updateRect()
	}
}

func (c *commandContext) SnapHorizontal() {
	if c.selectMode == selectModeInsert {
		return
	}
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
		c.selected[i] = m.TransformAffine(c.selected[i])
	}
	c.updateRect()
}

func (c *commandContext) SelectMatrix() (mat.Mat4, bool) {
	if c.selectMode == selectModeInsert {
		return mat.Mat4{}, false
	}
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

func (c *commandContext) baseFilter(selected bool) func(int, mat.Vec3) bool {
	if selected {
		return func(i int, p mat.Vec3) bool {
			mask := c.selectMask[i]
			return mask&selectBitmaskCropped == 0 && mask&selectBitmaskSelected != 0
		}
	}
	return func(i int, p mat.Vec3) bool {
		mask := c.selectMask[i]
		return mask&selectBitmaskCropped != 0 || mask&selectBitmaskSelected == 0
	}
}

func (c *commandContext) baseFilterByMask(selected bool) func(int, mat.Vec3) bool {
	if selected {
		return func(i int, p mat.Vec3) bool {
			mask := c.selectMask[i]
			return mask&selectBitmaskCropped == 0 && mask&selectBitmaskSegmentSelected != 0
		}
	}
	return func(i int, p mat.Vec3) bool {
		mask := c.selectMask[i]
		return mask&selectBitmaskCropped != 0 || mask&selectBitmaskSegmentSelected == 0
	}
}

func (c *commandContext) AddSurface(resolution float32) bool {
	if c.selectMode == selectModeInsert {
		return false
	}
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
	pcNew := &pc.PointCloud{
		PointCloudHeader: c.editor.pp.PointCloudHeader.Clone(),
		Points:           w * h,
		Data:             make([]byte, w*h*c.editor.pp.Stride()),
	}
	it, err := pcNew.Vec3Iterator()
	if err != nil {
		return false
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			it.SetVec3(
				m.TransformAffine(mat.Vec3{float32(x) * resolution, float32(y) * resolution, 0}),
			)
			it.Incr()
		}
	}
	c.editor.merge(pcNew)
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) Delete() bool {
	switch c.SelectMode() {
	case selectModeRect:
		filter := c.baseFilter(false) // keep unselected points
		c.editor.passThrough(filter)
		c.pointCloudUpdated = true
	case selectModeMask:
		c.editor.passThroughByMask(c.selectMask, selectBitmaskSegmentSelected, 0)
		c.selectMode = selectModeRect // selected points are deleted
		c.pointCloudUpdated = true
	}
	return true
}

func (c *commandContext) VoxelFilter(resolution float32) error {
	if c.SelectMode() != selectModeRect {
		return errors.New("VoxelFilter is not supported on segment based select")
	}

	var pp *pc.PointCloud

	_, selected := c.SelectMatrix()
	if selected {
		var err error
		if pp, err = passThrough(c.editor.pp, c.baseFilter(true)); err != nil {
			return err
		}
	} else {
		pp = c.editor.pp
	}

	vg := voxelgrid.New(mat.Vec3{resolution, resolution, resolution})
	pcFiltered, err := vg.Filter(pp)
	if err != nil {
		return err
	}

	if selected {
		c.editor.passThrough(c.baseFilter(false))
		c.editor.pop()
		c.editor.merge(pcFiltered)
	} else {
		if err := c.editor.SetPointCloud(pcFiltered, cloudMain); err != nil {
			return err
		}
	}

	c.pointCloudUpdated = true
	return nil
}

func (c *commandContext) Label(l uint32) bool {
	var filter func(int, mat.Vec3) bool
	switch c.SelectMode() {
	case selectModeRect:
		filter = c.baseFilter(true)
	case selectModeMask:
		filter = c.baseFilterByMask(true)
	default:
		return false
	}
	c.editor.label(func(i int, p mat.Vec3) (uint32, bool) {
		if filter(i, p) {
			return l, true
		}
		return 0, false
	})
	c.pointCloudUpdated = true
	return true
}

func (c *commandContext) Undo() bool {
	c.pointCloudUpdated = true
	return c.editor.Undo()
}

func (c *commandContext) MaxHistory() int {
	return c.editor.MaxHistory()
}

func (c *commandContext) SetMaxHistory(m int) bool {
	if m < 0 {
		return false
	}
	c.editor.SetMaxHistory(m)
	return true
}

func (c *commandContext) ImportPCD(blob interface{}) error {
	p, err := c.pcdIO.importPCD(blob)
	if err != nil {
		return err
	}
	if err := c.editor.SetPointCloud(p, cloudMain); err != nil {
		return err
	}

	c.pointCloudUpdated = true
	return nil
}

func (c *commandContext) ImportSubPCD(blob interface{}) error {
	if c.editor.pp == nil {
		return errors.New("must have base cloud")
	}
	p, err := c.pcdIO.importPCD(blob)
	if err != nil {
		return err
	}
	if err := c.editor.SetPointCloud(p, cloudSub); err != nil {
		return err
	}

	c.selectMode = selectModeInsert
	// Put unit vectors to reconstruct final transformation easily
	// by cursorsToTrans()
	c.selected = []mat.Vec3{
		{},
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	c.updateRect()

	c.subPointCloudUpdated = true
	return nil
}

func (c *commandContext) FinalizeCurrentMode() error {
	switch c.selectMode {
	case selectModeInsert:
		it, err := c.editor.ppSub.Vec3Iterator()
		if err != nil {
			return err
		}
		trans := cursorsToTrans(c.selected)
		for ; it.IsValid(); it.Incr() {
			it.SetVec3(trans.Transform(it.Vec3()))
		}
		c.editor.merge(c.editor.ppSub)
		c.pointCloudUpdated = true
		c.UnsetCursors()
	}
	return nil
}

func (c *commandContext) FitInserting(axes [6]bool) error {
	if c.selectMode != selectModeInsert {
		return errors.New("not in insert mode")
	}
	it, err := c.editor.pp.Vec3Iterator()
	if err != nil {
		return err
	}
	itSubOrig, err := c.editor.ppSub.Vec3Iterator()
	if err != nil {
		return err
	}
	trans := cursorsToTrans(c.selected)
	itSub := &transformedVec3RandomAccessor{
		Vec3RandomAccessor: itSubOrig,
		trans:              trans,
	}
	minMain, maxMain, err := pc.MinMaxVec3(it)
	if err != nil {
		return err
	}
	minSub, maxSub, err := pc.MinMaxVec3(itSub)
	if err != nil {
		return err
	}

	const (
		matchRange        = 0.5
		regionPadding     = 1.0
		maxBasePoints     = 60000
		maxTargetPoints   = 10000
		minSampleRatio    = 0.01
		gradientWeight    = 0.25
		gradientPosThresh = 0.001
		gradientRotThresh = 0.002
		maxIteration      = 50
	)

	is := rectIntersection(
		rect{minMain, maxMain},
		rect{minSub, maxSub},
	)
	is.min = is.min.Sub(mat.Vec3{regionPadding, regionPadding, regionPadding})
	is.max = is.max.Add(mat.Vec3{regionPadding, regionPadding, regionPadding})
	if !is.IsValid() {
		return errors.New("no intersection")
	}
	center := is.min.Add(is.max).Mul(0.5)

	sample := func(ra pc.Vec3RandomAccessor, isIn func(mat.Vec3) bool, nMax int) (pc.Vec3Slice, float32) {
		var cnt int
		for i := 0; i < ra.Len(); i++ {
			if isIn(ra.Vec3At(i)) {
				cnt++
			}
		}
		ratio := float32(nMax) / float32(cnt)
		out := make(pc.Vec3Slice, 0, nMax)
		for i := 0; i < ra.Len(); i++ {
			if p := ra.Vec3At(i); isIn(p) {
				if rand.Float32() > ratio {
					continue
				}
				out = append(out, p.Sub(center))
				if len(out) >= nMax {
					break
				}
			}
		}
		return out, ratio
	}

	base, ratioBase := sample(it, is.IsInside, maxBasePoints)
	if ratioBase < minSampleRatio {
		return errors.New("too many base points")
	}
	kdt := kdtree.New(base)

	// Sample points near the base cloud
	targetFilter := func(p mat.Vec3) bool {
		if !is.IsInside(p) {
			return false
		}
		id, _ := kdt.Nearest(p.Sub(center), regionPadding)
		return id >= 0
	}
	target, ratioTarget := sample(itSub, targetFilter, maxTargetPoints)
	if ratioTarget < minSampleRatio {
		return errors.New("too many inserting points")
	}

	gradientWeightVec := mat.Vec6{
		gradientWeight, gradientWeight, gradientWeight,
		gradientWeight, gradientWeight, gradientWeight,
	}
	for i, v := range axes {
		if !v {
			gradientWeightVec[i] = 0
		}
	}

	// Registration
	ppicp := &icp.PointToPointICPGradient{
		Evaluator: &icp.PointToPointEvaluator{
			Corresponder: &icp.NearestPointCorresponder{MaxDist: matchRange},
			MinPairs:     32,
			WeightFn: func(distSq float32) float32 {
				a := (1 - distSq/(matchRange*matchRange))
				return a * a
			},
		},
		MaxIteration:   maxIteration,
		GradientWeight: gradientWeightVec,
		GradientThreshold: mat.Vec6{
			gradientPosThresh, gradientPosThresh, gradientPosThresh,
			gradientRotThresh, gradientRotThresh, gradientRotThresh,
		},
	}
	transFit, stat, err := ppicp.Fit(kdt, target)
	if err != nil {
		return fmt.Errorf("registration failed: %v, stat: %v", err, stat)
	}
	println(fmt.Sprintf("%v", stat))

	transFit = mat.Translate(center[0], center[1], center[2]).
		Mul(transFit).
		Mul(mat.Translate(-center[0], -center[1], -center[2]))
	c.TransformCursors(transFit)

	return nil
}

func (c *commandContext) Import2D(yamlBlob, img interface{}) error {
	mi, imgJS, err := c.mapIO.readMap(yamlBlob, img)
	if err != nil {
		c.mapInfo = nil
		return err
	}
	c.mapInfo = mi
	c.mapImg = imgJS

	c.mapUpdated = true
	return nil
}

func (c *commandContext) ExportPCD() (interface{}, error) {
	if c.editor.pp == nil {
		return nil, errors.New("no pointcloud")
	}
	blob, err := c.pcdIO.exportPCD(c.editor.pp)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (c *commandContext) ExportSelectedPCD() (interface{}, error) {
	if c.editor.pp == nil {
		return nil, errors.New("no pointcloud")
	}
	var pp *pc.PointCloud
	var err error

	switch c.SelectMode() {
	case selectModeRect:
		pp, err = passThrough(c.editor.pp, c.baseFilter(true)) // extract selected
	case selectModeMask:
		pp, err = passThroughByMask(
			c.editor.pp, c.selectMask,
			selectBitmaskSegmentSelected, selectBitmaskSegmentSelected,
		)
	}
	if err != nil {
		return nil, err
	}

	if pp == nil || pp.Points == 0 {
		return nil, errors.New("no points are selected")
	}

	blob, err := c.pcdIO.exportPCD(pp)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (c *commandContext) SelectSegment(p mat.Vec3) {
	res := float32(c.segmentationDistance)
	w := int(c.segmentationRange / c.segmentationDistance)
	half := float32(w) * res / 2
	v := vgs.New(res, [3]int{w, w, w}, p.Sub(mat.Vec3{half, half, half}))

	it, err := c.editor.pp.Vec3Iterator()
	if err != nil {
		return
	}

	// Detect surface and exclude from selection.
	n := c.editor.pp.Points
	for i := 0; i < n; i++ {
		c.selectMask[i] &= ^uint32(selectBitmaskSegmentSelected)
		a, ok := v.Addr(it.Vec3())
		if ok {
			if c.selectMask[i]&(selectBitmaskCropped|selectBitmaskOnScreen) == selectBitmaskOnScreen {
				v.AddByAddr(a, i)
			}
		}
		it.Incr()
	}
	if it, err = c.editor.pp.Vec3Iterator(); err != nil {
		return
	}
	vIndice := v.Storage().Indice()
	raIn := pc.NewIndiceVec3RandomAccessor(it, vIndice)
	sacSurface := sac.New(
		sac.NewRandomSampler(raIn.Len()),
		sac.NewVoxelGridSurfaceModel(v.Storage(), raIn),
	)
	var surfRealIndice []int
	if ok := sacSurface.Compute(sacIterationCnt); ok {
		if coeff := sacSurface.Coefficients(); coeff.Evaluate() > sacSurfacePointsMin {
			if !coeff.IsIn(p, c.segmentationDistance) {
				// Excude points only if the selected point is not on the surface.
				surfIndice := coeff.Inliers(c.segmentationDistance)
				surfRealIndice := make([]int, len(surfIndice))
				for j, i := range surfIndice {
					ri := vIndice[i]
					c.selectMask[ri] |= selectBitmaskExclude
					surfRealIndice[j] = ri
				}
			}
		}
	}
	v.Reset()

	// Select segment start from clicked point.
	if it, err = c.editor.pp.Vec3Iterator(); err != nil {
		return
	}
	for _, i := range vIndice {
		if c.selectMask[i]&(selectBitmaskCropped|selectBitmaskOnScreen|selectBitmaskExclude) == selectBitmaskOnScreen {
			v.Add(it.Vec3At(i), i)
		}
	}

	for _, i := range v.Segment(p) {
		if c.selectMask[i]&selectBitmaskExclude == 0 {
			c.selectMask[i] |= selectBitmaskSegmentSelected
		}
	}
	for _, i := range surfRealIndice {
		// Clear selectBitmaskExclude bit.
		c.selectMask[i] &= 0xFFFFFFFF ^ uint32(selectBitmaskExclude)
	}
	c.UnsetCursors()
	c.selectMode = selectModeMask
}
