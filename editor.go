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
	pc         *pcd.PointCloud
	history    []*pcd.PointCloud
	maxHistory *int

	cropMatrix mat.Mat4
}

func (e *editor) MaxHistory() int {
	if e.maxHistory == nil {
		return maxHistoryDefault
	}
	return *e.maxHistory
}

func (e *editor) SetMaxHistory(m int) {
	if m < 0 {
		m = 0
	}
	e.maxHistory = &m
}

func (e *editor) push(pc *pcd.PointCloud) {
	shallowCopy := &pcd.PointCloud{
		PointCloudHeader: pc.PointCloudHeader.Clone(),
		Points:           pc.Points,
		Data:             pc.Data,
	}
	e.history = append(e.history, shallowCopy)
	if len(e.history) > e.MaxHistory()+1 {
		e.history[0] = nil
		e.history = e.history[1:]
		runtime.GC()
	}
	e.pc = pc
}

func (e *editor) pop() *pcd.PointCloud {
	back := e.history[len(e.history)-1]
	e.history = e.history[:len(e.history)-1]
	return back
}

func (e *editor) Crop(origin mat.Mat4) {
	e.cropMatrix = origin
}

func (e *editor) Undo() bool {
	if n := len(e.history); n > 1 {
		e.history[n-1] = nil
		e.history = e.history[:n-1]
		e.pc = e.history[n-2]
		return true
	}
	return false
}

func (e *editor) SetPointCloud(pc *pcd.PointCloud) error {
	if len(pc.Fields) == 4 && pc.Fields[0] == "x" && pc.Fields[1] == "y" && pc.Fields[2] == "z" && pc.Fields[3] == "label" {
		e.push(pc)
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
	e.push(pcNew)
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
	e.push(pcNew)
	return nil
}

func (e *editor) passThrough(fn func(int, mat.Vec3) bool) error {
	pc, err := passThrough(e.pc, fn)
	if err != nil {
		return err
	}
	e.push(pc)
	return nil
}

func passThrough(pc *pcd.PointCloud, fn func(int, mat.Vec3) bool) (*pcd.PointCloud, error) {
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
	i := 0
	for {
		var p mat.Vec3
		for it.IsValid() {
			p = it.Vec3()
			if fn(i, p) {
				break
			}
			i++
			it.Incr()
			itL.Incr()
		}
		if !it.IsValid() {
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

	e.push(pcNew)
}
