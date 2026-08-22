package main

import (
	"bytes"
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
