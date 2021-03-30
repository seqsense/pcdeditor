package main

import (
	"runtime"

	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

const (
	maxHistoryDefault = 4
)

type editor struct {
	history
	pc *pcd.PointCloud

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
	push(pc *pcd.PointCloud) *pcd.PointCloud
	pop() *pcd.PointCloud
	undo() (*pcd.PointCloud, bool)
}

func (e *editor) Undo() bool {
	var ok bool
	e.pc, ok = e.history.undo()
	return ok
}

func (e *editor) Crop(origin mat.Mat4) {
	e.cropMatrix = origin
}

func (e *editor) SetPointCloud(pc *pcd.PointCloud) error {
	if len(pc.Fields) == 4 && pc.Fields[0] == "x" && pc.Fields[1] == "y" && pc.Fields[2] == "z" && pc.Fields[3] == "label" {
		e.pc = e.push(pc)
		runtime.GC()
		return nil
	}
	i, err := pc.Vec3Iterator()
	if err != nil {
		return err
	}
	iL, _ := pc.Uint32Iterator("label")

	pcNew := &pcd.PointCloud{
		PointCloudHeader: pcd.PointCloudHeader{
			Version:   pc.Version,
			Fields:    []string{"x", "y", "z", "label"},
			Size:      []int{4, 4, 4, 4},
			Type:      []string{"F", "F", "F", "U"},
			Count:     []int{1, 1, 1, 1},
			Viewpoint: pc.Viewpoint,
			Width:     pc.Points,
			Height:    1,
		},
		Points: pc.Points,
	}
	pcNew.Data = make([]byte, pc.Points*pcNew.Stride())

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
	e.pc = e.push(pcNew)
	runtime.GC()
	return nil
}

func (e *editor) label(fn func(int, mat.Vec3) (uint32, bool)) error {
	pcNew := &pcd.PointCloud{
		PointCloudHeader: e.pc.PointCloudHeader.Clone(),
		Points:           e.pc.Points,
		Data:             make([]byte, len(e.pc.Data)),
	}
	copy(pcNew.Data, e.pc.Data)

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
	e.pc = e.push(pcNew)
	runtime.GC()
	return nil
}

func (e *editor) passThrough(fn func(int, mat.Vec3) bool) error {
	pc, err := passThrough(e.pc, fn)
	if err != nil {
		return err
	}
	e.pc = e.push(pc)
	runtime.GC()
	return nil
}

func (e *editor) passThroughByMask(sel []uint32, mask, val uint32) error {
	pc, err := passThroughByMask(e.pc, sel, mask, val)
	if err != nil {
		return err
	}
	e.pc = e.push(pc)
	runtime.GC()
	return nil
}

func passThrough(pc *pcd.PointCloud, fn func(int, mat.Vec3) bool) (*pcd.PointCloud, error) {
	return passThroughImpl(pc, func(it, jt pcd.Vec3Iterator, itL, jtL pcd.Uint32Iterator) int {
		i := 0
		n := pc.Points
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

func passThroughByMask(pc *pcd.PointCloud, sel []uint32, mask, val uint32) (*pcd.PointCloud, error) {
	return passThroughImpl(pc, func(it, jt pcd.Vec3Iterator, itL, jtL pcd.Uint32Iterator) int {
		i := 0
		n := pc.Points
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

func passThroughImpl(pc *pcd.PointCloud, core func(_, _ pcd.Vec3Iterator, _, _ pcd.Uint32Iterator) int) (*pcd.PointCloud, error) {
	pcNew := &pcd.PointCloud{
		PointCloudHeader: pc.PointCloudHeader.Clone(),
		Data:             make([]byte, len(pc.Data)),
		Points:           pc.Points,
	}

	it, err := pc.Vec3Iterator()
	if err != nil {
		return nil, err
	}
	jt, err := pcNew.Vec3Iterator()
	if err != nil {
		return nil, err
	}
	itL, err := pc.Uint32Iterator("label")
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

func (e *editor) merge(pc *pcd.PointCloud) {
	pcNew := &pcd.PointCloud{
		PointCloudHeader: e.pc.PointCloudHeader.Clone(),
		Points:           e.pc.Points + pc.Points,
		Data:             append(e.pc.Data[:e.pc.Stride()*e.pc.Points], pc.Data...),
	}
	pcNew.Width = pcNew.Points
	pcNew.Height = 1

	e.pc = e.push(pcNew)
	runtime.GC()
}
