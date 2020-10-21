package main

import (
	webgl "github.com/seqsense/pcdeditor/gl"
	"math"
)

const (
	defaultDistance = 100.0
	defaultPitch    = math.Pi / 4
	yDeadband       = 20
)

type view struct {
	fov              float64
	x, y, yaw, pitch float64
	distance         float64

	x0, y0, yaw0, pitch0 float64
	drag0                *webgl.MouseEvent
}

func newView() *view {
	return &view{
		distance: defaultDistance,
		pitch:    defaultPitch,
	}
}

func (v *view) reset() {
	v.distance = defaultDistance
	v.pitch = defaultPitch
}

func (v *view) fps() {
	v.distance = 0
	v.pitch = math.Pi / 2
}

func (v *view) snapYaw() {
	v.yaw = math.Round(v.yaw/(math.Pi/2)) * (math.Pi / 2)
}

func (v *view) snapPitch() {
	v.pitch = math.Round(v.pitch/(math.Pi/2)) * (math.Pi / 2)
}

func (v *view) wheel(e *webgl.WheelEvent) {
	v.distance += e.DeltaY * (v.distance*0.05 + 0.1)
	if v.distance < 0 {
		v.distance = 0
	} else if v.distance > 1000 {
		v.distance = 1000
	}
}

func (v *view) move(dx, dy, dyaw float64) {
	s, c := math.Sincos(v.yaw)
	v.x += c*dy + s*dx
	v.y += s*dy - c*dx
	v.yaw += dyaw
	v.yaw = math.Remainder(v.yaw, 2*math.Pi)
}

func (v *view) mouseDragStart(e *webgl.MouseEvent) {
	v.drag0 = e
	v.yaw0 = v.yaw
	v.pitch0 = v.pitch
	v.x0 = v.x
	v.y0 = v.y
}

func (v *view) mouseDragEnd(e *webgl.MouseEvent) {
	if v.drag0 == nil {
		return
	}
	v.mouseDrag(e)
	v.drag0 = nil
}

func (v *view) mouseDrag(e *webgl.MouseEvent) {
	if v.drag0 == nil {
		return
	}
	xDiff := float64(e.OffsetX - v.drag0.OffsetX)
	yDiff := float64(e.OffsetY - v.drag0.OffsetY)
	switch v.drag0.Button {
	case 0:
		v.yaw = v.yaw0 - 0.02*xDiff
		if yDiff < -yDeadband {
			yDiff += yDeadband
		} else if yDiff > yDeadband {
			yDiff -= yDeadband
		} else {
			yDiff = 0
		}
		v.pitch = v.pitch0 - 0.02*yDiff
		if v.pitch < 0 {
			v.pitch = 0
		} else if v.pitch > math.Pi {
			v.pitch = math.Pi
		}
	case 1:
		s, c := math.Sincos(v.yaw)
		v.x = v.x0 + 0.1*(xDiff*c+yDiff*s)
		v.y = v.y0 + 0.1*(xDiff*s-yDiff*c)
	}
}
