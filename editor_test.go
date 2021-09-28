package main

import (
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

func TestPassThrough(t *testing.T) {
	pp := &pc.PointCloud{
		PointCloudHeader: pc.PointCloudHeader{
			Fields: []string{"x", "y", "z", "label"},
			Size:   []int{4, 4, 4, 4},
			Type:   []string{"F", "F", "F", "U"},
			Count:  []int{1, 1, 1, 1},
			Width:  3,
			Height: 1,
		},
		Points: 3,
	}
	pp.Data = make([]byte, 3*pp.Stride())
	it, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	vecs := []mat.Vec3{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	it.SetVec3(vecs[0])
	it.Incr()
	it.SetVec3(vecs[1])
	it.Incr()
	it.SetVec3(vecs[2])

	check := func(t *testing.T, out *pc.PointCloud) {
		jt, err := out.Vec3Iterator()
		if err != nil {
			t.Fatal(err)
		}
		var outVec []mat.Vec3
		for ; jt.IsValid(); jt.Incr() {
			outVec = append(outVec, jt.Vec3())
		}
		expected := []mat.Vec3{
			vecs[0],
			vecs[2],
		}

		if !reflect.DeepEqual(expected, outVec) {
			t.Errorf("Expected:\n%v\nGot:\n%v", expected, outVec)
		}
	}
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
		check(t, out)
	})
	t.Run("ByMask", func(t *testing.T) {
		sel := []uint32{0x10, 0x111, 0x11}
		out, err := passThroughByMask(pp, sel, 0x110, 0x10)
		if err != nil {
			t.Fatal(err)
		}
		check(t, out)
	})
}
