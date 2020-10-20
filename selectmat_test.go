package main

import (
	"github.com/seqsense/pcdeditor/mat"
	"testing"
)

func TestMat4Filter(t *testing.T) {
	m := mat.Scale(2, 1, 1).MulAffine(mat.Translate(-2, -1, 0))
	f := mat4Filter(m)

	testCases := []struct {
		p        mat.Vec3
		expected bool
	}{
		{mat.NewVec3(0, 0, 0), true},
		{mat.NewVec3(2.1, 1.1, 0.1), false},
		{mat.NewVec3(2.4, 1.1, 0.1), false},
		{mat.NewVec3(2.6, 1.1, 0.1), true},
		{mat.NewVec3(2.1, 1.9, 0.1), false},
		{mat.NewVec3(2.1, 2.1, 0.1), true},
		{mat.NewVec3(2.1, 1.1, 0.9), false},
		{mat.NewVec3(2.1, 1.1, 1.1), true},
	}

	for i, tt := range testCases {
		res := f.Filter(tt.p)
		if res != tt.expected {
			t.Errorf(
				"[%d] Filter(%f, %f, %f) is expected to be %v",
				i, tt.p[0], tt.p[1], tt.p[2], tt.expected,
			)
		}
		resInv := f.FilterInv(tt.p)
		if resInv != (!tt.expected) {
			t.Errorf(
				"[%d] FilterInv(%f, %f, %f) is expected to be %v",
				i, tt.p[0], tt.p[1], tt.p[2], !tt.expected,
			)
		}
	}
}
