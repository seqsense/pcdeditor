package main

import (
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

func TestSelectPoint(t *testing.T) {
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
		{1, 2, 0},
		{3, 3, 0},
		{2, 2, 0},
	}
	it.SetVec3(vecs[0])
	it.Incr()
	it.SetVec3(vecs[1])
	it.Incr()
	it.SetVec3(vecs[2])

	testCases := map[string]struct {
		mask     []uint32
		x, y     float32
		selected bool
		expected mat.Vec3
	}{
		"Select0": {
			mask: []uint32{
				selectBitmaskNearCursor | selectBitmaskOnScreen,
				selectBitmaskNearCursor | selectBitmaskOnScreen,
				selectBitmaskNearCursor | selectBitmaskOnScreen,
			},
			x:        1,
			y:        2,
			selected: true,
			expected: mat.Vec3{1, 2, 0},
		},
		"Select2": {
			mask: []uint32{
				selectBitmaskNearCursor | selectBitmaskOnScreen,
				selectBitmaskNearCursor | selectBitmaskOnScreen,
				selectBitmaskNearCursor | selectBitmaskOnScreen,
			},
			x:        2,
			y:        2,
			selected: true,
			expected: mat.Vec3{2, 2, 0},
		},
		"NoValidPoint": {
			mask: []uint32{
				selectBitmaskOnScreen,
				selectBitmaskNearCursor,
				0,
			},
			x:        2,
			y:        2,
			selected: false,
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			model := mat.Translate(-tt.x, -tt.y, -10)
			proj := mat.Perspective(1.57, 1, 1, 100)
			p, ok := selectPoint(
				pp, tt.mask, ProjectionPerspective,
				&model,
				&proj,
				100, 100, 200, 200,
			)
			if !ok {
				if tt.selected {
					t.Fatal("Point must be selected")
				}
				return
			} else if !tt.selected {
				t.Fatal("Point must not be selected")
			}
			if !p.Equal(tt.expected) {
				t.Errorf("Expected %v to be selected, got %v", tt.expected, *p)
			}
		})
	}
}
