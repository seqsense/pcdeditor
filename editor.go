package main

import (
	"github.com/seqsense/pcdviewer/mat"
	"github.com/seqsense/pcdviewer/pcd"
)

type editor struct {
	pc *pcd.PointCloud
}

func (e *editor) Set(pc *pcd.PointCloud) error {
	if len(pc.Fields) == 4 && pc.Fields[0] == "x" && pc.Fields[1] == "y" && pc.Fields[2] == "z" && pc.Fields[3] == "label" {
		e.pc = pc
		return nil
	}
	i, err := pc.Vec3Iterator()
	if err != nil {
		return err
	}
	iL, _ := pc.Uint32Iterator("label")

	pcNew := &pcd.PointCloud{
		PointCloudHeader: pcd.PointCloudHeader{
			Version: pc.Version,
			Fields:  []string{"x", "y", "z", "label"},
			Size:    []int{4, 4, 4, 4},
			Type:    []string{"F", "F", "F", "U"},
			Count:   []int{1, 1, 1, 1},
			Width:   pc.Points,
			Height:  1,
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
	e.pc = pcNew
	return nil
}

func (e *editor) Label(fn func(mat.Vec3) (uint32, bool)) error {
	it, err := e.pc.Vec3Iterator()
	if err != nil {
		return err
	}
	itL, err := e.pc.Uint32Iterator("label")
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
	pcOld := e.pc
	e.pc = pcNew
	if len(indice) == 0 {
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
	iPrev := 0
	for _, i := range indice {
		for ; iPrev < i; iPrev++ {
			it.Incr()
		}
		jt.SetVec3(it.Vec3())
		jt.Incr()
	}
	return nil
}

func (e *editor) Merge(pc *pcd.PointCloud) {
	e.pc.Points += pc.Points
	e.pc.Width = e.pc.Points
	e.pc.Height = 1
	e.pc.Data = append(e.pc.Data, pc.Data...)
}
