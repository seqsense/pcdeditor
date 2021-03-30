package voxelgrid

import (
	"reflect"
	"testing"

	"github.com/seqsense/pcdeditor/mat"
)

func TestVoxelGrid(t *testing.T) {
	v := New(0.05, [3]int{64, 64, 64}, mat.Vec3{2, 5, 10})

	points := []mat.Vec3{
		{-2, 0, 0},
		{2, 5, 10},
		{2.01, 5, 10},
		{2 + 1, 5 + 1, 10 + 1},
		{2 + 3.21, 5, 10},
	}

	if v.Add(points[0], 0) {
		t.Error("Point out of the voxel grid should not be added")
	}
	if !v.Add(points[1], 1) {
		t.Error("Point in the voxel grid should be added")
	}
	if !v.Add(points[2], 2) {
		t.Error("Point in the voxel grid should be added")
	}
	if !v.Add(points[3], 3) {
		t.Error("Point in the voxel grid should be added")
	}
	if v.Add(points[4], 4) {
		t.Error("Point out of the voxel grid should be added")
	}

	if ids := v.Get(points[0]); ids != nil {
		t.Error("Point out of the voxel grid should not be added")
	}
	if ids := v.Get(points[1]); !reflect.DeepEqual([]int{1, 2}, ids) {
		t.Error("Points in the voxel differs")
	}
	if ids := v.Get(points[2]); !reflect.DeepEqual([]int{1, 2}, ids) {
		t.Error("Points in the voxel differs")
	}
	if ids := v.Get(points[3]); !reflect.DeepEqual([]int{3}, ids) {
		t.Error("Points in the voxel differs")
	}
	if ids := v.Get(points[4]); ids != nil {
		t.Error("Point out of the voxel grid should not be added")
	}
}
