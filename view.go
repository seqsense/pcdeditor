package main

type view interface {
	Reset()
	Fps()
	SnapYaw()
	SnapPitch()
	Move(dx, dy, dyaw float64)

	SetPitch(p float64)
	RotateYaw(y float64)
}
