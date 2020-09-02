package mat

import (
	"math"
)

func Translate(x, y, z float32) Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		x, y, z, 1,
	}
}

func Rotate(x, y, z, ang float32) Mat4 {
	s := float32(math.Sin(float64(ang)))
	c := float32(math.Cos(float64(ang)))

	return Mat4{
		c + x*x*(1-c), x*y*(1-c) - z*s, x*z*(1-c) + y*s, 0,
		y*x*(1-c) + z*s, c + y*y*(1-c), y*z*(1-c) - x*s, 0,
		z*x*(1-c) - y*s, z*y*(1-c) + x*s, c * z * z * (1 - c), 0,
		0, 0, 0, 1,
	}
}
