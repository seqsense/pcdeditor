package voxelgrid

import (
	"github.com/seqsense/pcdeditor/mat"
	storage "github.com/seqsense/pcdeditor/pcd/storage/voxelgrid"
)

const initialSliceCap = 8192

var cursor [][3]int

func init() {
	for _, x := range []int{-1, 0, 1} {
		for _, y := range []int{-1, 0, 1} {
			for _, z := range []int{-1, 0, 1} {
				if x == 0 && y == 0 && z == 0 {
					continue
				}
				cursor = append(cursor, [3]int{x, y, z})
			}
		}
	}
}

type VoxelGrid struct {
	*storage.VoxelGrid
}

func New(resolution float32, size [3]int, origin mat.Vec3) *VoxelGrid {
	return &VoxelGrid{
		VoxelGrid: storage.New(resolution, size, origin),
	}
}

func (v *VoxelGrid) Segment(p mat.Vec3) []int {
	searched := make([]bool, v.Len())
	pos, ok := v.PosInt(p)
	if !ok {
		return nil
	}
	next := make([][3]int, 0, initialSliceCap)
	next = append(next, pos)
	indice := make([]int, 0, initialSliceCap)

	for len(next) > 0 {
		var pos [3]int
		pos, next = next[0], next[1:]
		addr, ok := v.AddrByPosInt(pos)
		if !ok || searched[addr] {
			continue
		}
		searched[addr] = true
		c := v.GetByAddr(addr)
		if len(c) == 0 {
			continue
		}
		indice = append(indice, c...)

		for _, d := range cursor {
			n := [3]int{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
			addr, ok := v.AddrByPosInt(n)
			if !ok || searched[addr] {
				continue
			}
			next = append(next, n)
		}
	}
	return indice
}
