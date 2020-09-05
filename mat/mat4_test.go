package mat

import (
	"testing"
)

func TestMul(t *testing.T) {
	m0 := Translate(0.1, 0.2, 0.3)
	m1 := Scale(1.1, 1.2, 1.3)
	m2 := Rotate(1, 0, 0, 0.1)
	m3 := Rotate(0, 1, 0, 0.1)
	m4 := Rotate(0, 0, 1, 0.1)

	r := m0.MulAffine(m1).MulAffine(m2).MulAffine(m3).MulAffine(m4)
	rNaive := m0.Mul(m1).Mul(m2).Mul(m3).Mul(m4)

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			a := j*4 + i
			diff := r[a] - rNaive[a]
			if diff < -0.01 || 0.01 < diff {
				t.Errorf("m(%d, %d) expected to be %0.3f, got %0.3f",
					i, j, rNaive[a], r[a],
				)
			}
		}
	}
}

func TestInv(t *testing.T) {
	m0 := Translate(0.1, 0.2, 0.3)
	m1 := Scale(1.1, 1.2, 1.3)
	m2 := Rotate(1, 0, 0, 0.5)

	m := m0.MulAffine(m1).MulAffine(m2)
	mi := m.InvAffine()

	diag := m.Mul(mi)
	for i := 0; i < 4; i++ {
		t.Logf("%+0.1f %+0.1f %+0.1f %+0.1f", diag[4*i+0], diag[4*i+1], diag[4*i+2], diag[4*i+3])
		for j := 0; j < 3; j++ {
			if i == j {
				if diag[4*i+j] < 0.99 || 1.01 < diag[4*i+j] {
					t.Errorf("m(%d, %d): %0.3f", i, j, diag[4*i+j])
				}
			} else {
				if diag[4*i+j] < -0.01 || 0.01 < diag[4*i+j] {
					t.Errorf("m(%d, %d): %0.3f", i, j, diag[4*i+j])
				}
			}
		}
	}
}

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
