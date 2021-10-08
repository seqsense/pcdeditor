package main

import (
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

var vecs = []mat.Vec3{
	{1, 2, 3},
	{4, 5, 6},
	{7, 8, 9},
}

var labels = []uint32{0, 1, 2}
var intensities = []float32{0.5, 0.5, 0.5}

func createPointCloud(t *testing.T, withIntensity bool) *pc.PointCloud {
	var header pc.PointCloudHeader
	if withIntensity {
		header = pc.PointCloudHeader{
			Fields: []string{"x", "y", "z", "intensity", "label"},
			Size:   []int{4, 4, 4, 4, 4},
			Type:   []string{"F", "F", "F", "F", "U"},
			Count:  []int{1, 1, 1, 1, 1},
			Width:  len(vecs),
			Height: 1,
		}
	} else {
		header = pc.PointCloudHeader{
			Fields: []string{"x", "y", "z", "label"},
			Size:   []int{4, 4, 4, 4},
			Type:   []string{"F", "F", "F", "U"},
			Count:  []int{1, 1, 1, 1},
			Width:  len(vecs),
			Height: 1,
		}
	}
	pp := &pc.PointCloud{
		PointCloudHeader: header,
		Points:           len(vecs),
	}
	pp.Data = make([]byte, len(vecs)*pp.Stride())
	vt, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	lt, err := pp.Uint32Iterator("label")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(vecs); i++ {
		vt.SetVec3(vecs[i])
		lt.SetUint32(labels[i])
		vt.Incr()
		lt.Incr()
	}

	if withIntensity {
		it, err := pp.Float32Iterator("intensity")
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(intensities); i++ {
			it.SetFloat32(intensities[i])
			it.Incr()
		}
	}

	return pp
}

func check(t *testing.T, out *pc.PointCloud, indices []int) {
	vt, err := out.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	lt, err := out.Uint32Iterator("label")
	if err != nil {
		t.Fatal(err)
	}

	var expectedVec []mat.Vec3
	var expectedLabel []uint32
	for _, i := range indices {
		expectedVec = append(expectedVec, vecs[i])
		expectedLabel = append(expectedLabel, labels[i])
	}

	var outVec []mat.Vec3
	for ; vt.IsValid(); vt.Incr() {
		outVec = append(outVec, vt.Vec3())
	}
	if !reflect.DeepEqual(expectedVec, outVec) {
		t.Errorf("Expected:\n%v\nGot:\n%v", expectedVec, outVec)
	}

	var outLabel []uint32
	for ; lt.IsValid(); lt.Incr() {
		outLabel = append(outLabel, lt.Uint32())
	}
	if !reflect.DeepEqual(expectedLabel, outLabel) {
		t.Errorf("Expected:\n%v\nGot:\n%v", expectedLabel, outLabel)
	}
}

func TestPassThrough(t *testing.T) {
	pp := createPointCloud(t, false)

	indices := []int{0, 2}
	t.Run("ByVec", func(t *testing.T) {
		out, err := passThrough(pp, func(i int, v mat.Vec3) bool {
			if !v.Equal(vecs[i]) {
				t.Errorf("Expected %v, got %v", vecs[i], v)
			}
			return i != 1
		})
		if err != nil {
			t.Fatal(err)
		}
		check(t, out, indices)
	})
	t.Run("ByMask", func(t *testing.T) {
		sel := []uint32{0x10, 0x111, 0x11}
		out, err := passThroughByMask(pp, sel, 0x110, 0x10)
		if err != nil {
			t.Fatal(err)
		}
		check(t, out, indices)
	})
}

func TestSetPointCloud(t *testing.T) {
	e := newEditor()
	pps := []*pc.PointCloud{
		createPointCloud(t, false),
		createPointCloud(t, true),
	}
	indices := []int{0, 1, 2}

	for _, pp := range pps {
		if err := e.SetPointCloud(pp); err != nil {
			t.Fatal(err)
		}
		check(t, e.pp, indices)
	}
}
