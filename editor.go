package main

import (
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

const (
	maxHistory = 5
)

type editor struct {
	pc      *pcd.PointCloud
	history []*pcd.PointCloud
}

func (e *editor) push(pc *pcd.PointCloud) {
	shallowCopy := &pcd.PointCloud{
		PointCloudHeader: pc.PointCloudHeader.Clone(),
		Points:           pc.Points,
		Data:             pc.Data,
	}
	e.history = append(e.history, shallowCopy)
	if len(e.history) > maxHistory {
		e.history[0] = nil
		e.history = e.history[1:]
	}
	e.pc = pc
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

func (e *editor) Set(pc *pcd.PointCloud) error {
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
	for i.IsValid() {
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

func (e *editor) Label(fn func(mat.Vec3) (uint32, bool)) error {
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

	for it.IsValid() {
		l, ok := fn(it.Vec3())
		if ok {
			itL.SetUint32(l)
		}
		it.Incr()
		itL.Incr()
	}
	e.push(pcNew)
	return nil
}

func (e *editor) Filter(fn func(mat.Vec3) bool) error {
	it, err := e.pc.Vec3Iterator()
	if err != nil {
		return err
	}

	indice := make([]int, 0, e.pc.Points)
	for i := 0; it.IsValid(); {
		if fn(it.Vec3()) {
			indice = append(indice, i)
		}
		i++
		it.Incr()
	}
	pcNew := &pcd.PointCloud{
		PointCloudHeader: e.pc.PointCloudHeader.Clone(),
		Points:           len(indice),
		Data:             make([]byte, len(indice)*e.pc.Stride()),
	}
	pcNew.Width = len(indice)
	pcNew.Height = 1

	pcOld := e.pc
	if len(indice) == 0 {
		e.push(pcNew)
		return nil
	}
	it, err = pcOld.Vec3Iterator()
	if err != nil {
		return err
	}
	jt, err := pcNew.Vec3Iterator()
	if err != nil {
		return err
	}
	itL, err := pcOld.Uint32Iterator("label")
	if err != nil {
		return err
	}
	jtL, err := pcNew.Uint32Iterator("label")
	if err != nil {
		return err
	}
	iPrev := 0
	for _, i := range indice {
		for ; iPrev < i; iPrev++ {
			it.Incr()
			itL.Incr()
		}
		jt.SetVec3(it.Vec3())
		jtL.SetUint32(itL.Uint32())
		jt.Incr()
		jtL.Incr()
	}
	e.push(pcNew)
	return nil
}

func (e *editor) Merge(pc *pcd.PointCloud) {
	e.pc.Points += pc.Points
	e.pc.Width = e.pc.Points
	e.pc.Height = 1
	e.pc.Data = append(e.pc.Data, pc.Data...)
	e.push(e.pc)
}
