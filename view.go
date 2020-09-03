package main

import (
	webgl "github.com/seqsense/pcdviewer/gl"
	"math"
)

type view struct {
	fov       float64
	x, y, ang float64

	xStart, yStart, angStart float64
	dragStart                *webgl.MouseEvent
}

func (v *view) mouseDragStart(e *webgl.MouseEvent) {
	v.dragStart = e
	v.angStart = v.ang
	v.xStart = v.x
	v.yStart = v.y
}

func (v *view) mouseDragEnd(e *webgl.MouseEvent) {
	v.mouseDrag(e)
	v.dragStart = nil
}

func (v *view) mouseDrag(e *webgl.MouseEvent) {
	if v.dragStart == nil {
		return
	}
	xDiff := float64(e.ClientX - v.dragStart.ClientX)
	yDiff := float64(e.ClientY - v.dragStart.ClientY)
	switch v.dragStart.Button {
	case 0:
		v.ang = v.angStart - 0.02*xDiff
	case 1:
		s, c := math.Sincos(v.ang)
		v.x = v.xStart + 0.1*(xDiff*c+yDiff*s)
		v.y = v.yStart + 0.1*(xDiff*s-yDiff*c)
	}
}
