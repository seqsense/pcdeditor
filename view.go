package main

type view interface {
	Reset()
	FPS()
	SnapYaw()
	SnapPitch()
	Move(dx, dy, dyaw float64)

	View() [5]float64
	SetView([5]float64) error

	SetPitch(p float64)
	RotateYaw(y float64)

	IncreaseFOV()
	DecreaseFOV()
}
