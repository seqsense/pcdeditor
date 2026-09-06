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
			Version:   0.7,
			Fields:    []string{"x", "y", "z", "label"},
			Size:      []int{4, 4, 4, 4},
			Type:      []string{"F", "F", "F", "U"},
			Count:     []int{1, 1, 1, 1},
			Width:     width,
			Height:    height,
			Viewpoint: []float32{0, 0, 0, 1, 0, 0, 0},
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
		t.Fatal("Data mismatch after restore")
	}
}

func TestSavedLabelsRestore(t *testing.T) {
	orig := makeTestCloud(t, 100, 100, 1)
	pp := cloneCloud(orig)

	stride := pp.Stride()
	p := &savedLabels{}
	for _, i := range []uint32{0, 3, 42, 99} {
		off := int(i)*stride + 12
		p.Indices = append(p.Indices, i)
		p.OldLabels = append(p.OldLabels, binary.LittleEndian.Uint32(pp.Data[off:]))
		binary.LittleEndian.PutUint32(pp.Data[off:], 12345)
	}

	out, err := p.restore(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
}

func deleteForTest(pp *pc.PointCloud, removed map[int]bool) *removedPoints {
	stride := pp.Stride()
	p := &removedPoints{
		OldWidth:  pp.Width,
		OldHeight: pp.Height,
	}
	j := 0
	for i := 0; i < pp.Points; i++ {
		if removed[i] {
			p.Indices = append(p.Indices, uint32(i))
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

func TestRemovedPointsRestore(t *testing.T) {
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
				out, err := p.restore(pp)
				if err != nil {
					t.Fatal(err)
				}
				assertCloudEqual(t, orig, out)
			})
			t.Run("NoSpareCapacity", func(t *testing.T) {
				pp := cloneCloud(orig)
				p := deleteForTest(pp, removed)
				// Drop the spare capacity to exercise the reallocation
				// path (a no-op for the None pattern).
				pp.Data = append([]byte{}, pp.Data...)
				out, err := p.restore(pp)
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

func TestPreviousSizeRestore(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	pp := cloneCloud(orig)

	p := &previousSize{Points: pp.Points, Width: pp.Width, Height: pp.Height}
	added := makeTestCloud(t, 10, 10, 1)
	pp.Data = append(pp.Data, added.Data...)
	pp.Points += added.Points
	pp.Width = pp.Points
	pp.Height = 1

	out, err := p.restore(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
}

func TestPreviousCloudRestore(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	pp := makeTestCloud(t, 5, 5, 1)

	p := newPreviousCloud(orig)
	out, err := p.restore(pp)
	if err != nil {
		t.Fatal(err)
	}
	assertCloudEqual(t, orig, out)
	if !reflect.DeepEqual(orig.PointCloudHeader, out.PointCloudHeader) {
		t.Fatalf("Header: expected %+v, got %+v", orig.PointCloudHeader, out.PointCloudHeader)
	}
}

func TestRecordEncodeDecodeRoundTrip(t *testing.T) {
	orig := makeTestCloud(t, 100, 10, 10)
	orig.Viewpoint = []float32{1, 2, 3, 1, 0, 0, 0}
	for name, d := range map[string]undoData{
		"PreviousCloud": newPreviousCloud(orig),
		"SavedLabels":   &savedLabels{Indices: []uint32{1, 2, 42}, OldLabels: []uint32{7, 8, 9}},
		"PreviousSize":  &previousSize{Points: 90, Width: 9, Height: 10},
		"RemovedPoints": &removedPoints{
			OldWidth: 10, OldHeight: 10,
			Indices: []uint32{0, 50, 99},
			points:  bytes.Repeat([]byte{1, 2, 3, 4}, 3*4),
		},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := encodeUndoData(&buf, d); err != nil {
				t.Fatal(err)
			}
			buf.Write(d.payload())
			decoded, err := decodeRecord(buf.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(d, decoded) {
				t.Errorf("Expected %+v, got %+v", d, decoded)
			}
		})
	}
}
