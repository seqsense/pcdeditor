package voxelgrid

import (
	"github.com/seqsense/pcdeditor/mat"
)

type VoxelGrid struct {
	voxel         [][]int
	size          [3]int
	origin        mat.Vec3
	resolution    float32
	resolutionInv float32
}

func New(resolution float32, size [3]int, origin mat.Vec3) *VoxelGrid {
	return &VoxelGrid{
		voxel:         make([][]int, size[0]*size[1]*size[2]),
		size:          size,
		origin:        origin,
		resolution:    resolution,
		resolutionInv: 1 / resolution,
	}
}

func (v *VoxelGrid) MinMax() (min, max mat.Vec3) {
	return v.origin, v.origin.Add(mat.Vec3{
		float32(v.size[0]) * v.resolution,
		float32(v.size[1]) * v.resolution,
		float32(v.size[2]) * v.resolution,
	})
}

func (v *VoxelGrid) Resolution() float32 {
	return v.resolution
}

func (v *VoxelGrid) Add(p mat.Vec3, index int) bool {
	addr, ok := v.Addr(p)
	if !ok {
		return false
	}
	ptr := &v.voxel[addr]
	*ptr = append(*ptr, index)
	return true
}

func (v *VoxelGrid) Get(p mat.Vec3) []int {
	addr, ok := v.Addr(p)
	if !ok {
		return nil
	}
	return v.voxel[addr]
}

func (v *VoxelGrid) GetByAddr(a int) []int {
	return v.voxel[a]
}

func (v *VoxelGrid) Addr(p mat.Vec3) (int, bool) {
	pos := p.Sub(v.origin)
	x := int(pos[0]*v.resolutionInv + 0.5)
	if x < 0 || x >= v.size[0] {
		return 0, false
	}
	y := int(pos[1]*v.resolutionInv + 0.5)
	if y < 0 || y >= v.size[1] {
		return 0, false
	}
	z := int(pos[2]*v.resolutionInv + 0.5)
	if z < 0 || z >= v.size[2] {
		return 0, false
	}
	return x + (y+z*v.size[1])*v.size[0], true
}

func (v *VoxelGrid) AddrByPosInt(p [3]int) (int, bool) {
	x, y, z := p[0], p[1], p[2]
	if x < 0 || y < 0 || z < 0 || x >= v.size[0] || y >= v.size[1] || z >= v.size[2] {
		return 0, false
	}
	return x + (y+z*v.size[1])*v.size[0], true
}

func (v *VoxelGrid) PosInt(p mat.Vec3) ([3]int, bool) {
	pos := p.Sub(v.origin)
	x := int(pos[0]*v.resolutionInv + 0.5)
	if x < 0 || x >= v.size[0] {
		return [3]int{}, false
	}
	y := int(pos[1]*v.resolutionInv + 0.5)
	if y < 0 || y >= v.size[1] {
		return [3]int{}, false
	}
	z := int(pos[2]*v.resolutionInv + 0.5)
	if z < 0 || z >= v.size[2] {
		return [3]int{}, false
	}
	return [3]int{x, y, z}, true
}

func (v *VoxelGrid) Len() int {
	return v.size[0] * v.size[1] * v.size[2]
}

func (v *VoxelGrid) Indice() []int {
	out := make([]int, 0, 1024)
	for _, g := range v.voxel {
		out = append(out, g...)
	}
	return out
}

func (v *VoxelGrid) Reset() {
	for i := range v.voxel {
		v.voxel[i] = nil
	}
}
