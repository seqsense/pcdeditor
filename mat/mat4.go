package mat

type Mat4 [16]float32

func (m Mat4) Add(a Mat4) Mat4 {
	var out Mat4
	for i := range m {
		out[i] = m[i] + a[i]
	}
	return out
}

func (m Mat4) Mul(a Mat4) Mat4 {
	var out Mat4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var sum float32
			for k := 0; k < 4; k++ {
				sum += m[4*k+i] * a[4*j+k]
			}
			out[4*j+i] = sum
		}
	}
	return out
}
