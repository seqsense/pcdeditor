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
	pp *pc.PointCloud

	cropMatrix mat.Mat4
}

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
}

func (e *editor) Undo() bool {
	pp, ok := e.history.undo()
	if ok {
		e.pp = pp
	}
	return ok
}

func (e *editor) Crop(origin mat.Mat4) {
	e.cropMatrix = origin
}

func (e *editor) SetPointCloud(pp *pc.PointCloud) error {
	if len(pp.Fields) == 4 && pp.Fields[0] == "x" && pp.Fields[1] == "y" && pp.Fields[2] == "z" && pp.Fields[3] == "label" {
		e.pp = e.push(pp)
		runtime.GC()
		return nil
	}
	i, err := pp.Vec3Iterator()
	if err != nil {
		return err
	}
	iL, _ := pp.Uint32Iterator("label")

	pcNew := &pc.PointCloud{
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
			jL.SetUint32(jL.Uint32())
			jL.Incr()
			iL.Incr()
		}
	}
	e.pp = e.push(pcNew)
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

func passThrough(pp *pc.PointCloud, fn func(int, mat.Vec3) bool) (*pc.PointCloud, error) {
	return passThroughImpl(pp, func(it, jt pc.Vec3Iterator, itL, jtL pc.Uint32Iterator) int {
		i := 0
		n := pp.Points
		for {
			var p mat.Vec3
			for i < n {
				p = it.Vec3()
				if fn(i, p) {
					break
				}
				i++
				it.Incr()
				itL.Incr()
			}
			if i >= n {
				break
			}
			jt.SetVec3(p)
			jtL.SetUint32(itL.Uint32())
			jt.Incr()
			jtL.Incr()
			it.Incr()
			itL.Incr()
			i++
		}
		return i
	})
}

func passThroughByMask(pp *pc.PointCloud, sel []uint32, mask, val uint32) (*pc.PointCloud, error) {
	return passThroughImpl(pp, func(it, jt pc.Vec3Iterator, itL, jtL pc.Uint32Iterator) int {
		i := 0
		n := pp.Points
		for {
			for i < n {
				if sel[i]&mask == val {
					break
				}
				i++
				it.Incr()
				itL.Incr()
			}
			if i >= n {
				break
			}
			jt.SetVec3(it.Vec3())
			jtL.SetUint32(itL.Uint32())
			jt.Incr()
			jtL.Incr()
			it.Incr()
			itL.Incr()
			i++
		}
		return i
	})
}

func passThroughImpl(pp *pc.PointCloud, core func(_, _ pc.Vec3Iterator, _, _ pc.Uint32Iterator) int) (*pc.PointCloud, error) {
	pcNew := &pc.PointCloud{
		PointCloudHeader: pp.PointCloudHeader.Clone(),
		Data:             make([]byte, len(pp.Data)),
		Points:           pp.Points,
	}

	it, err := pp.Vec3Iterator()
	if err != nil {
		return nil, err
	}
	jt, err := pcNew.Vec3Iterator()
	if err != nil {
		return nil, err
	}
	itL, err := pp.Uint32Iterator("label")
	if err != nil {
		return nil, err
	}
	jtL, err := pcNew.Uint32Iterator("label")
	if err != nil {
		return nil, err
	}
	i := core(it, jt, itL, jtL)

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
