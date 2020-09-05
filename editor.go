package main

import (
	"github.com/seqsense/pcdviewer/mat"
	"github.com/seqsense/pcdviewer/pcd"
)

type editor struct {
	pc *pcd.PointCloud
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
}
