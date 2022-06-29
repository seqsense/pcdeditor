package main

import (
	"runtime"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

const (
	maxHistoryDefault = 4
)

type editor struct {
	history
	pp        *pc.PointCloud
	ppSub     *pc.PointCloud
	ppSubRect rect

	cropMatrix mat.Mat4
}

type cloudID int

const (
	cloudMain cloudID = iota
	cloudSub
)

func newEditor() *editor {
	return &editor{
		history: newHistory(maxHistoryDefault),
	}
}

type history interface {
	MaxHistory() int
	SetMaxHistory(m int)
	push(pp *pc.PointCloud) *pc.PointCloud
	pop() *pc.PointCloud
	undo() (*pc.PointCloud, bool)
	clear()
}

func (e *editor) Undo() bool {
	pp, ok := e.history.undo()
	if ok {
		e.pp = pp
	}
	return ok
}

func (e *editor) Reset() {
	e.clear()
	e.pp = nil
	e.ppSub = nil
	e.ppSubRect = rect{}
	e.cropMatrix = mat.Mat4{}
}

func (e *editor) Crop(origin mat.Mat4) {
	e.cropMatrix = origin
}

func (e *editor) SetPointCloud(pp *pc.PointCloud, id cloudID) error {
	if pp == nil && id == cloudSub {
		e.ppSub = nil
		runtime.GC()
		return nil
	}
	var pcNew *pc.PointCloud
	if len(pp.Fields) == 4 && pp.Fields[0] == "x" && pp.Fields[1] == "y" && pp.Fields[2] == "z" && pp.Fields[3] == "label" {
		pcNew = pp
	} else {
		i, err := pp.Vec3Iterator()
		if err != nil {
			return err
		}
		iL, _ := pp.Uint32Iterator("label")

		pcNew = &pc.PointCloud{
			PointCloudHeader: pc.PointCloudHeader{
				Version:   pp.Version,
				Fields:    []string{"x", "y", "z", "label"},
				Size:      []int{4, 4, 4, 4},
				Type:      []string{"F", "F", "F", "U"},
				Count:     []int{1, 1, 1, 1},
				Viewpoint: pp.Viewpoint,
				Width:     pp.Points,
				Height:    1,
			},
			Points: pp.Points,
		}
		pcNew.Data = make([]byte, pp.Points*pcNew.Stride())

		j, err := pcNew.Vec3Iterator()
		if err != nil {
			return err
		}
		jL, err := pcNew.Uint32Iterator("label")
		if err != nil {
			return err
		}
		for i.IsValid() && j.IsValid() {
			j.SetVec3(i.Vec3())
			j.Incr()
			i.Incr()
			if iL != nil {
				jL.SetUint32(iL.Uint32())
				jL.Incr()
				iL.Incr()
			}
		}
	}
	switch id {
	case cloudMain:
		e.pp = e.push(pcNew)
	case cloudSub:
		e.ppSub = pcNew
		it, err := pcNew.Vec3Iterator()
		if err != nil {
			return err
		}
		min, max, err := pc.MinMaxVec3(it)
		if err != nil {
			return err
		}
		e.ppSubRect = rect{
			min: min,
			max: max,
		}
	}
	runtime.GC()
	return nil
}

func (e *editor) label(fn func(int, mat.Vec3) (uint32, bool)) error {
	pcNew := &pc.PointCloud{
		PointCloudHeader: e.pp.PointCloudHeader.Clone(),
		Points:           e.pp.Points,
		Data:             make([]byte, len(e.pp.Data)),
	}
	copy(pcNew.Data, e.pp.Data)

	it, err := pcNew.Vec3Iterator()
	if err != nil {
		return err
	}
	itL, err := pcNew.Uint32Iterator("label")
	if err != nil {
		return err
	}

	i := 0
	for it.IsValid() {
		l, ok := fn(i, it.Vec3())
		if ok {
			itL.SetUint32(l)
		}
		it.Incr()
		itL.Incr()
		i++
	}
	e.pp = e.push(pcNew)
	runtime.GC()
	return nil
}

func (e *editor) passThrough(fn func(int, mat.Vec3) bool) error {
	pp, err := passThrough(e.pp, fn)
	if err != nil {
		return err
	}
	e.pp = e.push(pp)
	runtime.GC()
	return nil
}

func (e *editor) passThroughByMask(sel []uint32, mask, val uint32) error {
	pp, err := passThroughByMask(e.pp, sel, mask, val)
	if err != nil {
		return err
	}
	e.pp = e.push(pp)
	runtime.GC()
	return nil
}

func (e *editor) relabelPointsInLabelRange(minLabel, maxLabel, newLabel uint32) error {
	_, err := e.pp.Uint32Iterator("label")
	if err != nil {
		return err
	}

	pcNew := &pc.PointCloud{
		PointCloudHeader: e.pp.PointCloudHeader.Clone(),
		Data:             make([]byte, len(e.pp.Data)),
		Points:           e.pp.Points,
	}
	copy(pcNew.Data, e.pp.Data)
	pcNew.Width = e.pp.Width
	pcNew.Height = e.pp.Height

	lt, err := pcNew.Uint32Iterator("label")
	if err != nil {
		return err
	}

	for ; lt.IsValid(); lt.Incr() {
		l := lt.Uint32()
		if l == newLabel || l < minLabel || l > maxLabel {
			continue
		}
		lt.SetUint32(newLabel)
	}

	e.pp = e.push(pcNew)
	runtime.GC()
	return nil
}

func (e *editor) unlabelPoints(labelsToKeep []uint32) error {
	_, err := e.pp.Uint32Iterator("label")
	if err != nil {
		return err
	}

	pcNew := &pc.PointCloud{
		PointCloudHeader: e.pp.PointCloudHeader.Clone(),
		Data:             make([]byte, len(e.pp.Data)),
		Points:           e.pp.Points,
	}
	copy(pcNew.Data, e.pp.Data)
	pcNew.Width = e.pp.Width
	pcNew.Height = e.pp.Height

	lt, err := pcNew.Uint32Iterator("label")
	if err != nil {
		return err
	}

	isInLabelsToKeep := func(l uint32) bool {
		for _, kl := range labelsToKeep {
			if kl == l {
				return true
			}
		}
		return false
	}

	for ; lt.IsValid(); lt.Incr() {
		if isInLabelsToKeep(lt.Uint32()) {
			continue
		}
		lt.SetUint32(0)
	}

	e.pp = e.push(pcNew)
	runtime.GC()
	return nil
}

func passThrough(pp *pc.PointCloud, fn func(int, mat.Vec3) bool) (*pc.PointCloud, error) {
	it, err := pp.Vec3Iterator()
	if err != nil {
		return nil, err
	}
	return passThroughImpl(pp, func(dst, src *pc.PointCloud) int {
		i, j := 0, 0
		is, js, cnt := 0, 0, 0
		n := pp.Points
		for {
			for {
				if i >= n {
					if cnt > 0 {
						pc.Copy(dst, js, src, is, cnt)
					}
					return j
				}
				p := it.Vec3()
				if fn(i, p) {
					break
				}
				it.Incr()
				i++
				if cnt > 0 {
					pc.Copy(dst, js, src, is, cnt)
					cnt = 0
				}
			}
			if cnt == 0 {
				is, js = i, j
			}
			it.Incr()
			i++
			j++
			cnt++
		}
	})
}

func passThroughByMask(pp *pc.PointCloud, sel []uint32, mask, val uint32) (*pc.PointCloud, error) {
	return passThroughImpl(pp, func(dst, src *pc.PointCloud) int {
		i, j := 0, 0
		is, js, cnt := 0, 0, 0
		n := pp.Points
		for {
			for {
				if i >= n {
					if cnt > 0 {
						pc.Copy(dst, js, src, is, cnt)
					}
					return j
				}
				if sel[i]&mask == val {
					break
				}
				i++
				if cnt > 0 {
					pc.Copy(dst, js, src, is, cnt)
					cnt = 0
				}
			}
			if cnt == 0 {
				is, js = i, j
			}
			i++
			j++
			cnt++
		}
	})
}

func passThroughImpl(pp *pc.PointCloud, core func(_, _ *pc.PointCloud) int) (*pc.PointCloud, error) {
	pcNew := &pc.PointCloud{
		PointCloudHeader: pp.PointCloudHeader.Clone(),
		Data:             make([]byte, len(pp.Data)),
		Points:           pp.Points,
	}

	i := core(pcNew, pp)

	pcNew.Points = i
	pcNew.Width = i
	pcNew.Height = 1
	pcNew.Data = pcNew.Data[: i*pcNew.Stride() : i*pcNew.Stride()]
	return pcNew, nil
}

func (e *editor) merge(pp *pc.PointCloud) {
	pcNew := &pc.PointCloud{
		PointCloudHeader: e.pp.PointCloudHeader.Clone(),
		Points:           e.pp.Points + pp.Points,
		Data:             append(e.pp.Data[:e.pp.Stride()*e.pp.Points], pp.Data...),
	}
	pcNew.Width = pcNew.Points
	pcNew.Height = 1

	e.pp = e.push(pcNew)
	runtime.GC()
}
