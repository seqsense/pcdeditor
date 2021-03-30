package voxelgrid

import (
	"reflect"
	"sort"
	"testing"

	"github.com/seqsense/pcdeditor/mat"
)

func TestVoxelGrid_GetSegment(t *testing.T) {
	pc := []mat.Vec3{
		{0.00, 0.00, 0.00}, // 0
		{0.10, 0.00, 0.00}, // 1
		{0.50, 0.50, 0.50}, // 2
		{0.10, 0.05, 0.00}, // 3
		{0.51, 0.51, 0.50}, // 4
		{0.49, 0.51, 0.49}, // 5
		{0.45, 0.40, 0.45}, // 6
		{0.53, 0.50, 0.50}, // 7
		{0.57, 0.50, 0.50}, // 8
		{0.60, 0.50, 0.50}, // 9
		{0.80, 0.50, 0.50}, // 10
		{0.45, 0.45, 0.45}, // 11
		{0.48, 0.43, 0.48}, // 12
		{0.51, 0.47, 0.50}, // 13
	}

	expected := []int{2, 4, 5, 6, 7, 8, 9, 11, 12, 13}

	v := New(0.05, [3]int{64, 64, 64}, mat.Vec3{0.5 - 0.05*32, 0.5 - 0.05*32, 0.5 - 0.05*32})
	for i, p := range pc {
		v.Add(p, i)
	}
	indice := v.Segment(mat.Vec3{0.5, 0.5, 0.5})
	sort.Ints(indice)
	if !reflect.DeepEqual(expected, indice) {
		t.Errorf("Expected indice:\n%v\ngot:\n%v", expected, indice)
	}
}
