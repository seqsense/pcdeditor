package mat

import (
	"testing"
)

func TestCross(t *testing.T) {
	x := Vec3{1, 0, 0}
	y := Vec3{0, 1, 0}

	c := x.Cross(y)
	if c[0] < -0.01 || 0.01 < c[0] {
		t.Error("Cross()[0] is wrong")
	}
	if c[1] < -0.01 || 0.01 < c[1] {
		t.Error("Cross()[1] is wrong")
	}
	if c[2] < 0.99 || 1.01 < c[2] {
		t.Error("Cross()[2] is wrong")
	}

	cn := x.CrossNormSq(y)
	if cn < 0.99 || 1.01 < cn {
		t.Error("CrossNormSq is wrong")
	}
}
