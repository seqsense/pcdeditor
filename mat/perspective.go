package mat

import (
	"math"
)

func Perspective(fov, aspect, near, far float32) Mat4 {
	halfFovCot := 1 / float32(math.Tan(float64(fov/2)))
	return Mat4{
		halfFovCot, 0, 0, 0,
		0, aspect * halfFovCot, 0, 0,
		0, 0, -(far + near) / (far - near), -1,
		0, 0, -2 * far * near / (far - near), 0,
	}
}
