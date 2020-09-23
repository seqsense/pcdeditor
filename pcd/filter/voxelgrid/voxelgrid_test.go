package voxelgrid

import (
	"math"
	"testing"

	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
	"github.com/seqsense/pcdeditor/pcd/internal/float"
)

func TestVoxelGrid(t *testing.T) {
	pc := pcd.PointCloud{
		PointCloudHeader: pcd.PointCloudHeader{
			Fields: []string{"x", "y", "z", "label"},
			Size:   []int{4, 4, 4, 4},
			Count:  []int{1, 1, 1, 1},
			Width:  5,
			Height: 1,
		},
		Points: 5,
		Data: float.Float32SliceAsByteSlice([]float32{
			0.50, 1.50, 0.10, math.Float32frombits(1),
			1.00, 1.00, 1.00, math.Float32frombits(2),
			0.52, 1.50, 0.12, math.Float32frombits(3),
			1.00, 0.00, 1.00, math.Float32frombits(4),
			1.00, 1.02, 1.00, math.Float32frombits(5),
		}),
	}

	vg := New(mat.Vec3{0.1, 0.1, 0.1})
	out, err := vg.Filter(&pc)
	if err != nil {
		t.Fatal(err)
	}

	expected := []mat.Vec3{
		{0.51, 1.50, 0.11},
		{1.00, 0.00, 1.00},
		{1.00, 1.01, 1.00},
	}
	expectedLabels := []uint32{
		1, 4, 2,
	}

	if len(expected) != out.Points {
		t.Fatalf("Wrong number of points, expected: %d, got: %d", len(expected), out.Points)
	}
	it, err := out.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	lt, err := out.Uint32Iterator("label")
	if err != nil {
		t.Fatal(err)
	}

	for i, e := range expected {
		p := it.Vec3()
		if !p.Equal(e) {
			t.Errorf("Expected point: %v, got: %v", e, p)
		}
		l := lt.Uint32()
		if l != expectedLabels[i] {
			t.Errorf("Expected label: %x, got: %x", expectedLabels[i], l)
		}
		it.Incr()
		lt.Incr()
	}
}
