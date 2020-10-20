package main

import (
	"github.com/seqsense/pcdeditor/mat"
)

type mat4Filter mat.Mat4

func (m mat4Filter) Filter(p mat.Vec3) bool {
	if z := mat.Mat4(m).TransformZ(p); z < 0 || 1 < z {
		return true
	}
	if x := mat.Mat4(m).TransformX(p); x < 0 || 1 < x {
		return true
	}
	if y := mat.Mat4(m).TransformY(p); y < 0 || 1 < y {
		return true
	}
	return false
}

func (m mat4Filter) FilterInv(p mat.Vec3) bool {
	if z := mat.Mat4(m).TransformZ(p); z < 0 || 1 < z {
		return false
	}
	if x := mat.Mat4(m).TransformX(p); x < 0 || 1 < x {
		return false
	}
	if y := mat.Mat4(m).TransformY(p); y < 0 || 1 < y {
		return false
	}
	return true
}
