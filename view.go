package main

type view interface {
	Reset()
	FPS()
	SnapYaw()
	SnapPitch()
	Move(dx, dy, dyaw float64)

	View() (x, y, yaw, pitch, distance float64)
	SetView(x, y, yaw, pitch, distance float64) error

	SetPitch(p float64)
	RotateYaw(y float64)

	IncreaseFOV()
	DecreaseFOV()
}
