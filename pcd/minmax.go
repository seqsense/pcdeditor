package pcd

import (
	"errors"
	"math"

	"github.com/seqsense/pcdeditor/mat"
)

func MinMaxVec3(pc *PointCloud) (mat.Vec3, mat.Vec3, error) {
	it, err := pc.Vec3Iterator()
	if err != nil {
		return mat.Vec3{}, mat.Vec3{}, err
	}
	if !it.IsValid() {
		return mat.Vec3{}, mat.Vec3{}, errors.New("no point")
	}
	min := mat.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	max := mat.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	for ; it.IsValid(); it.Incr() {
		v := it.Vec3()
		for i := range v {
			if v[i] < min[i] {
				min[i] = v[i]
			}
			if v[i] > max[i] {
				max[i] = v[i]
			}
		}
	}
	return min, max, nil
}
