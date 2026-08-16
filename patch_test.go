package main

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/pc"
)

func makeTestCloud(t *testing.T, n, width, height int) *pc.PointCloud {
	t.Helper()
	pp := &pc.PointCloud{
		PointCloudHeader: pc.PointCloudHeader{
			Version: 0.7,
			Fields:  []string{"x", "y", "z", "label"},
			Size:    []int{4, 4, 4, 4},
			Type:    []string{"F", "F", "F", "U"},
			Count:   []int{1, 1, 1, 1},
			Width:   width,
			Height:  height,
		},
		Points: n,
	}
	pp.Data = make([]byte, n*pp.Stride())
	rnd := rand.New(rand.NewSource(int64(n)))
	rnd.Read(pp.Data)
	return pp
}

func cloneCloud(pp *pc.PointCloud) *pc.PointCloud {
	out := &pc.PointCloud{
		PointCloudHeader: pp.PointCloudHeader.Clone(),
		Points:           pp.Points,
		Data:             append([]byte{}, pp.Data...),
	}
	return out
}

func assertCloudEqual(t *testing.T, expected, got *pc.PointCloud) {
	t.Helper()
	if expected.Points != got.Points {
		t.Fatalf("Points: expected %d, got %d", expected.Points, got.Points)
	}
	if expected.Width != got.Width || expected.Height != got.Height {
		t.Fatalf("Size: expected %dx%d, got %dx%d",
			expected.Width, expected.Height, got.Width, got.Height)
	}
	if !bytes.Equal(expected.Data, got.Data) {
		t.Fatal("Data mismatch after revert")
	}
}

func TestLabelPatchRevert(t *testing.T) {
	orig := makeTestCloud(t, 100, 100, 1)
	pp := cloneCloud(orig)

	stride := pp.Stride()
	p := &labelPatch{}
	for _, i := range []uint32{0, 3, 42, 99} {
		off := int(i)*stride + 12
		p.indices = append(p.indices, i)
		p.oldLabels = append(p.oldLabels, binary.LittleEndian.Uint32(pp.Data[off:]))
		binary.LittleEndian.PutUint32(pp.Data[off:], 12345)
	}

	out, err := p.revert(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
}

func deleteForTest(pp *pc.PointCloud, removed map[int]bool) *deletePatch {
	stride := pp.Stride()
	p := &deletePatch{
		oldWidth:  pp.Width,
		oldHeight: pp.Height,
	}
	j := 0
	for i := 0; i < pp.Points; i++ {
		if removed[i] {
			p.indices = append(p.indices, uint32(i))
			p.points = append(p.points, pp.Data[i*stride:(i+1)*stride]...)
			continue
		}
		if i != j {
			copy(pp.Data[j*stride:(j+1)*stride], pp.Data[i*stride:(i+1)*stride])
		}
		j++
	}
	pp.Points = j
	pp.Width = j
	pp.Height = 1
	pp.Data = pp.Data[:j*stride]
	return p
}

func TestDeletePatchRevert(t *testing.T) {
	for name, removed := range map[string]map[int]bool{
		"Scattered": {1: true, 5: true, 6: true, 99: true},
		"Head":      {0: true, 1: true, 2: true},
		"Tail":      {97: true, 98: true, 99: true},
		"All":       allIndices(100),
		"None":      {},
	} {
		t.Run(name, func(t *testing.T) {
			orig := makeTestCloud(t, 100, 10, 10)
			t.Run("KeptCapacity", func(t *testing.T) {
				pp := cloneCloud(orig)
				p := deleteForTest(pp, removed)
				out, err := p.revert(pp)
				if err != nil {
					t.Fatal(err)
				}
				assertCloudEqual(t, orig, out)
			})
			t.Run("Realloc", func(t *testing.T) {
				pp := cloneCloud(orig)
				p := deleteForTest(pp, removed)
				// Force the reallocation path by dropping spare capacity.
				pp.Data = append([]byte{}, pp.Data...)
				out, err := p.revert(pp)
				if err != nil {
					t.Fatal(err)
				}
				assertCloudEqual(t, orig, out)
			})
		})
	}
}

func allIndices(n int) map[int]bool {
	m := map[int]bool{}
	for i := 0; i < n; i++ {
		m[i] = true
	}
	return m
}

func TestAppendPatchRevert(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	pp := cloneCloud(orig)

	p := &appendPatch{oldPoints: pp.Points, oldWidth: pp.Width, oldHeight: pp.Height}
	added := makeTestCloud(t, 10, 10, 1)
	pp.Data = append(pp.Data, added.Data...)
	pp.Points += added.Points
	pp.Width = pp.Points
	pp.Height = 1

	out, err := p.revert(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
}

func TestReplacePatchRevert(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	orig.Viewpoint = []float32{0, 0, 0, 1, 0, 0, 0}
	pp := makeTestCloud(t, 5, 5, 1)

	p := &replacePatch{
		header: orig.PointCloudHeader.Clone(),
		data:   append([]byte{}, orig.Data...),
	}
	out, err := p.revert(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
	if !reflect.DeepEqual(orig.PointCloudHeader, out.PointCloudHeader) {
		t.Fatalf("Header: expected %+v, got %+v", orig.PointCloudHeader, out.PointCloudHeader)
	}
}

func TestPatchEncodeDecodeRoundTrip(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	orig.Viewpoint = []float32{1, 2, 3, 1, 0, 0, 0}
	patches := []patch{
		&labelPatch{indices: []uint32{1, 2, 42}, oldLabels: []uint32{7, 8, 9}},
		&deletePatch{
			oldWidth: 10, oldHeight: 10,
			indices: []uint32{0, 50, 99},
			points:  bytes.Repeat([]byte{1, 2, 3, 4}, 3*4),
		},
		&appendPatch{oldPoints: 90, oldWidth: 9, oldHeight: 10},
		&replacePatch{header: orig.PointCloudHeader.Clone(), data: orig.Data},
	}

	var buf bytes.Buffer
	encodePatches(&buf, patches)
	decoded, err := decodePatches(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(patches) {
		t.Fatalf("Expected %d patches, got %d", len(patches), len(decoded))
	}
	for i := range patches {
		if !reflect.DeepEqual(patches[i], decoded[i]) {
			t.Errorf("Patch %d: expected %+v, got %+v", i, patches[i], decoded[i])
		}
	}
}
