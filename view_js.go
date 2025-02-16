package main

import (
	webgl "github.com/seqsense/webgl-go"
	"math"
)

const (
	defaultDistance = 100.0
	defaultPitch    = math.Pi / 4
	defaultFOV      = math.Pi / 3
	fovMin          = math.Pi / 8
	fovMax          = math.Pi * 2 / 3
	fovUnit         = math.Pi / 16
	yDeadband       = 20
)

type viewImpl struct {
	fov              float32
	x, y, yaw, pitch float64
	distance         float64

	x0, y0, yaw0, pitch0 float64
	drag0                *webgl.MouseEvent
}

func newView() *viewImpl {
	return &viewImpl{
		distance: defaultDistance,
		pitch:    defaultPitch,
		fov:      defaultFOV,
	}
}

func (v *viewImpl) Reset() {
	v.yaw = 0
	v.distance = defaultDistance
	v.pitch = defaultPitch
	v.fov = defaultFOV
}

func (v *viewImpl) FPS() {
	v.distance = 0
	v.pitch = math.Pi / 2
	v.fov = fovMax
}

func (v *viewImpl) SnapYaw() {
	v.yaw = math.Round(v.yaw/(math.Pi/2)) * (math.Pi / 2)
}

func (v *viewImpl) SnapPitch() {
	v.pitch = math.Round(v.pitch/(math.Pi/2)) * (math.Pi / 2)
}

func (v *viewImpl) RotateYaw(y float64) {
	v.yaw += y
}

func (v *viewImpl) SetPitch(p float64) {
	v.pitch = p
}

func (v *viewImpl) Move(dx, dy, dyaw float64) {
	s, c := math.Sincos(v.yaw)
	v.x += c*dy + s*dx
	v.y += s*dy - c*dx
	v.yaw += dyaw
	v.yaw = math.Remainder(v.yaw, 2*math.Pi)
}

func (v *viewImpl) IncreaseFOV() {
	v.setFOV(v.fov + fovUnit)
}

func (v *viewImpl) DecreaseFOV() {
	v.setFOV(v.fov - fovUnit)
}

func (v *viewImpl) setFOV(fov float32) {
	switch {
	case fov < fovMin:
		fov = fovMin
	case fov > fovMax:
		fov = fovMax
	}
	v.fov = fov
}

func (v *viewImpl) wheel(e *webgl.WheelEvent) {
	v.distance += e.DeltaY * (v.distance*0.05 + 0.1)
	if v.distance < 0 {
		v.distance = 0
	} else if v.distance > 1000 {
		v.distance = 1000
	}
}

func (v *viewImpl) dragging() bool {
	return v.drag0 != nil
}

func (v *viewImpl) mouseDragStart(e *webgl.MouseEvent) {
	v.drag0 = e
	v.yaw0 = v.yaw
	v.pitch0 = v.pitch
	v.x0 = v.x
	v.y0 = v.y
}

func (v *viewImpl) mouseDragEnd(e *webgl.MouseEvent) {
	if v.drag0 == nil {
		return
	}
	v.mouseDrag(e)
	v.drag0 = nil
}

func (v *viewImpl) mouseDrag(e *webgl.MouseEvent) {
	if v.drag0 == nil {
		return
	}
	xDiff := float64(e.OffsetX - v.drag0.OffsetX)
	yDiff := float64(e.OffsetY - v.drag0.OffsetY)

	type dragType int
	const (
		dragRotate dragType = iota
		dragTranslate
	)

	var t dragType
	switch v.drag0.Button {
	case 0: // Left
		if e.ShiftKey {
			t = dragTranslate
		} else {
			t = dragRotate
		}
	case 1: // Middle
		t = dragTranslate
	}

	switch t {
	case dragRotate:
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
	case dragTranslate:
		s, c := math.Sincos(v.yaw)
		v.x = v.x0 + 0.1*(xDiff*c+yDiff*s)
		v.y = v.y0 + 0.1*(xDiff*s-yDiff*c)
	}
}

func (v *viewImpl) View() (x, y, yaw, pitch, distance float64) {
	return v.x, v.y, v.yaw, v.pitch, v.distance
}
func (v *viewImpl) SetView(x, y, yaw, pitch, distance float64) error {
	v.x, v.y, v.yaw, v.pitch, v.distance = x, y, yaw, pitch, distance
	return nil
}
