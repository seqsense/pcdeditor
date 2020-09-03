package mat

import (
	"testing"
)

func transformNaive(m Mat4, a Vec3) Vec3 {
	var out Vec3
	in := [4]float32{a[0], a[1], a[2], 1}
	for i := 0; i < 3; i++ {
		var sum float32
		for k := 0; k < 4; k++ {
			sum += m[4*k+i] * in[k]
		}
		out[i] = sum
	}
	return out
}

func TestTransform(t *testing.T) {
	m0 := Translate(0.1, 0.2, 0.3)
	m1 := Scale(1.1, 1.2, 1.3)
	m2 := Rotate(1, 0, 0, 0.1)
	m3 := Rotate(0, 1, 0, 0.1)
	m4 := Rotate(0, 0, 1, 0.1)

	m := m0.Mul(m1).Mul(m2).Mul(m3).Mul(m4)

	in := NewVec3(1, 2, 3)
	v := m.Transform(in)
	vNaive := transformNaive(m, in)

	for i := 0; i < 3; i++ {
		diff := v[i] - vNaive[i]
		if diff < -0.01 || 0.01 < diff {
			t.Errorf("v(%d) expected to be %0.3f, got %0.3f",
				i, vNaive[i], v[i],
			)
		}
	}
}
