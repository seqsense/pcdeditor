package pcd

import (
	"errors"
)

type PointCloudHeader struct {
	Version   float32
	Fields    []string
	Size      []int
	Type      []string
	Count     []int
	Width     int
	Height    int
	Viewpoint []float32
}

func (h *PointCloudHeader) Clone() PointCloudHeader {
	return PointCloudHeader{
		Version:   h.Version,
		Fields:    append([]string{}, h.Fields...),
		Size:      append([]int{}, h.Size...),
		Type:      append([]string{}, h.Type...),
		Count:     append([]int{}, h.Count...),
		Width:     h.Width,
		Height:    h.Height,
		Viewpoint: append([]float32{}, h.Viewpoint...),
	}
}

type PointCloud struct {
	PointCloudHeader
	Points int

	Data      []byte
	dataFloat []float32
}

func (pc *PointCloudHeader) Stride() int {
	var stride int
	for i := range pc.Fields {
		stride += pc.Count[i] * pc.Size[i]
	}
	return stride
}

func (pc *PointCloud) Float32Iterator(name string) (Float32Iterator, error) {
	offset := 0
	for i, fn := range pc.Fields {
		if fn == name {
			if pc.Stride()&3 == 0 && offset&3 == 0 {
				// Aligned
				if pc.dataFloat == nil {
					pc.dataFloat = byteSliceAsFloat32Slice(pc.Data)
				}
				return &float32Iterator{
					data:   pc.dataFloat,
					pos:    offset / 4,
					stride: pc.Stride() / 4,
				}, nil
			}
			return &binaryFloat32Iterator{
				binaryIterator: binaryIterator{
					data:   pc.Data,
					pos:    offset,
					stride: pc.Stride(),
				},
			}, nil
		}
		offset += pc.Size[i] * pc.Count[i]
	}
	return nil, errors.New("invalid field name")
}

func (pc *PointCloud) Float32Iterators(names ...string) ([]Float32Iterator, error) {
	var its []Float32Iterator
	for _, name := range names {
		it, err := pc.Float32Iterator(name)
		if err != nil {
			return nil, err
		}
		its = append(its, it)
	}
	return its, nil
}

func (pc *PointCloud) Vec3Iterator() (Vec3Iterator, error) {
	var xyz int
	for _, name := range pc.Fields {
		if name == "x" && xyz == 0 {
			xyz = 1
		} else if name == "y" && xyz == 1 {
			xyz = 2
		} else if name == "z" && xyz == 2 {
			xyz = 3
		}
	}
	if xyz != 3 {
		return pc.naiveVec3Iterator()
	}
	it, err := pc.Float32Iterator("x")
	if err != nil {
		return nil, err
	}
	vit, ok := it.(*float32Iterator)
	if !ok {
		return pc.naiveVec3Iterator()
	}
	return vit, nil
}

func (pc *PointCloud) naiveVec3Iterator() (Vec3Iterator, error) {
	its, err := pc.Float32Iterators("x", "y", "z")
	if err != nil {
		return nil, err
	}
	return naiveVec3Iterator{its[0], its[1], its[2]}, nil
}
