package sac

import (
	"reflect"
	"testing"

	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd/storage/voxelgrid"
)

func TestSAC(t *testing.T) {
	pc := dummyPointCloud{
		mat.Vec3{0.0, 0.0, 0.0},
		mat.Vec3{0.1, 0.0, 0.1},
		mat.Vec3{0.2, 0.0, 0.2},
		mat.Vec3{0.2, 0.1, 0.6}, // outlier
		mat.Vec3{0.0, 0.1, 0.0},
		mat.Vec3{0.1, 0.1, 0.1},
		mat.Vec3{0.2, 0.1, 0.2},
		mat.Vec3{0.0, 0.2, 0.0},
		mat.Vec3{0.1, 0.2, 0.1},
		mat.Vec3{0.2, 0.2, 0.2},
		mat.Vec3{0.3, 0.7, 0.0}, // outlier
		mat.Vec3{0.6, 0.7, 0.0}, // outlier
		mat.Vec3{0.6, 0.3, 0.0}, // outlier
	}
	vg := voxelgrid.New(0.1, [3]int{8, 8, 8}, mat.Vec3{})
	for i, p := range pc {
		vg.Add(p, i)
	}
	m := NewVoxelGridSurfaceModel(vg, pc)

	s := New(NewRandomSampler(len(pc)), m)
	if ok := s.Compute(30); !ok {
		t.Fatal("SAC.Compute should succeed")
	}

	indice := s.Coefficients().Inliers(0.1)
	expectedIndice := []int{0, 1, 2, 4, 5, 6, 7, 8, 9}
	if !reflect.DeepEqual(expectedIndice, indice) {
		t.Errorf("Expected inlier: %v, got: %v", expectedIndice, indice)
	}
}
