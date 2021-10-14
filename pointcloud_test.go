package main

import (
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

func TestTransformedVec3RandomAccessor(t *testing.T) {
	in := pc.Vec3Slice{
		{1, 2, 3},
		{2, 3, 4},
		{3, 4, 5},
	}
	ra := &transformedVec3RandomAccessor{
		Vec3RandomAccessor: in,
		trans:              mat.Translate(1, -2, -4),
	}

	expected := pc.Vec3Slice{
		{2, 0, -1},
		{3, 1, 0},
		{4, 2, 1},
	}
	if ra.Len() != in.Len() {
		t.Fatalf("Input and output length must be same, in: %d, out: %d", in.Len(), ra.Len())
	}

	for i, e := range expected {
		v := ra.Vec3At(i)
		if !e.Equal(v) {
			t.Errorf("Expected Vec3At(%d): %v, got: %v", i, e, v)
		}
	}
}

func TestRectIntersection(t *testing.T) {
	testCases := map[string]struct {
		a, b     rect
		expected rect
	}{
		"ABottomRight": {
			a:        rect{mat.Vec3{1, 2, 3}, mat.Vec3{5, 6, 7}},
			b:        rect{mat.Vec3{4, 5, 6}, mat.Vec3{7, 8, 9}},
			expected: rect{mat.Vec3{4, 5, 6}, mat.Vec3{5, 6, 7}},
		},
		"Mixed": {
			a:        rect{mat.Vec3{1, 2, 3}, mat.Vec3{5, 6, 7}},
			b:        rect{mat.Vec3{4, 3, 2}, mat.Vec3{6, 4, 10}},
			expected: rect{mat.Vec3{4, 3, 3}, mat.Vec3{5, 4, 7}},
		},
		"NoOverlap": {
			a:        rect{mat.Vec3{1, 2, 3}, mat.Vec3{3, 4, 5}},
			b:        rect{mat.Vec3{6, 7, 8}, mat.Vec3{9, 10, 11}},
			expected: rect{mat.Vec3{6, 7, 8}, mat.Vec3{3, 4, 5}},
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Run("Forward", func(t *testing.T) {
				out := rectIntersection(tt.a, tt.b)
				if !reflect.DeepEqual(tt.expected, out) {
					t.Errorf("Expected rect: %v, got: %v", tt.expected, out)
				}
			})
			t.Run("Reverse", func(t *testing.T) {
				out := rectIntersection(tt.b, tt.a)
				if !reflect.DeepEqual(tt.expected, out) {
					t.Errorf("Expected rect: %v, got: %v", tt.expected, out)
				}
			})
		})
	}
}

func TestRect(t *testing.T) {
	type insideCheck struct {
		p      mat.Vec3
		inside bool
	}

	testCases := map[string]struct {
		r           rect
		valid       bool
		insideCheck map[string]insideCheck
	}{
		"Valid": {
			r:     rect{mat.Vec3{4, 5, 6}, mat.Vec3{5, 6, 7}},
			valid: true,
			insideCheck: map[string]insideCheck{
				"Inside": {
					p:      mat.Vec3{4.5, 5.6, 6.7},
					inside: true,
				},
				"Outside1": {
					p:      mat.Vec3{3.5, 5.6, 6.7},
					inside: false,
				},
				"Outside2": {
					p:      mat.Vec3{5.5, 5.6, 6.7},
					inside: false,
				},
				"Outside3": {
					p:      mat.Vec3{4.5, 6.6, 6.7},
					inside: false,
				},
				"Outside4": {
					p:      mat.Vec3{4.5, 5.6, 7.7},
					inside: false,
				},
			},
		},
		"InValid": {
			r:     rect{mat.Vec3{6, 7, 8}, mat.Vec3{3, 4, 5}},
			valid: false,
			insideCheck: map[string]insideCheck{
				"BetweenRects": {
					p:      mat.Vec3{4, 5, 6},
					inside: false,
				},
				"Outside": {
					p:      mat.Vec3{10, 10, 10},
					inside: false,
				},
			},
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			ok := tt.r.IsValid()
			if ok != tt.valid {
				if tt.valid {
					t.Error("Expected to be valid")
				} else {
					t.Error("Expected to be invalid")
				}
			}
			for name, ic := range tt.insideCheck {
				ic := ic
				t.Run(name, func(t *testing.T) {
					inside := tt.r.IsInside(ic.p)
					if inside != ic.inside {
						if ic.inside {
							t.Errorf("%v is expected to be inside", ic.p)
						} else {
							t.Errorf("%v is expected to be outside", ic.p)
						}
					}
				})
			}
		})
	}
}
