package main

import (
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

func TestSelectRange(t *testing.T) {
	c := newCommandContext(nil, nil)
	c.SetProjectionType(ProjectionPerspective)

	c.SetSelectRange(rangeTypeAuto, 123)
	if v := c.SelectRange(rangeTypeAuto); v != 123 {
		t.Errorf("SelectRangeAuto must be updated, expected: 123, got: %f", v)
	}

	c.SetSelectRange(rangeTypePerspective, 124)
	if v := c.SelectRange(rangeTypeAuto); v != 124 {
		t.Errorf("SelectRangeAuto must be updated by setting rangeTypePerspective, expected: 124, got: %f", v)
	}
	if v := c.SelectRange(rangeTypePerspective); v != 124 {
		t.Errorf("SelectRangePerspective must be updated, expected: 124, got: %f", v)
	}

	c.SetSelectRange(rangeTypeOrtho, 125)
	if v := c.SelectRange(rangeTypeAuto); v != 124 {
		t.Errorf("SelectRangeAuto must not be updated by setting rangeTypeOrtho, expected: 124, got: %f", v)
	}
	if v := c.SelectRange(rangeTypeOrtho); v != 125 {
		t.Errorf("SelectRangeOrtho must be updated, expected: 125, got: %f", v)
	}

	c.SetProjectionType(ProjectionOrthographic)
	if v := c.SelectRange(rangeTypeAuto); v != 125 {
		t.Errorf("SelectRangeAuto must not be updated by setting rangeTypeOrtho, expected: 125, got: %f", v)
	}
}

func expectPointCloud(t *testing.T, pp *pc.PointCloud, vecs []mat.Vec3) {
	t.Helper()
	it, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != it.Len() {
		t.Fatalf("Expected %d points, has %d points", len(vecs), it.Len())
	}
	for i, v := range vecs {
		if !v.Equal(it.Vec3()) {
			t.Fatalf("Expected %v, got %v at %d", v, it.Vec3(), i)
		}
		it.Incr()
	}
}

func TestImportPCD(t *testing.T) {
	header := pc.PointCloudHeader{
		Fields: []string{"x", "y", "z"},
		Size:   []int{4, 4, 4},
		Type:   []string{"F", "F", "F"},
		Count:  []int{1, 1, 1},
		Width:  1,
		Height: 1,
	}
	pp0 := &pc.PointCloud{
		PointCloudHeader: header,
		Points:           1,
		Data:             make([]byte, 4*3),
	}
	it0, err := pp0.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	it0.SetVec3(mat.Vec3{1, 2, 3})

	pp1 := &pc.PointCloud{
		PointCloudHeader: header,
		Points:           1,
		Data:             make([]byte, 4*3),
	}
	it1, err := pp1.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	it1.SetVec3(mat.Vec3{4, 5, 6})

	t.Run("ImportPCD", func(t *testing.T) {
		c := newCommandContext(&dummyPCDIO{}, nil)
		if err := c.ImportPCD(pp0); err != nil {
			t.Fatal(err)
		}

		out, _, hasOut := c.PointCloud()
		if !hasOut {
			t.Fatal("PointCloud is not stored")
		}
		expectPointCloud(t, out, []mat.Vec3{{1, 2, 3}})
	})
	t.Run("ImportSubPCD", func(t *testing.T) {
		c := newCommandContext(&dummyPCDIO{}, nil)
		if err := c.ImportPCD(pp0); err != nil {
			t.Fatal(err)
		}
		if err := c.ImportSubPCD(pp1); err != nil {
			t.Fatal(err)
		}

		outSub0, _, hasOutSub0 := c.SubPointCloud()
		if !hasOutSub0 {
			t.Fatal("Sub PointCloud is not stored")
		}
		expectPointCloud(t, outSub0, []mat.Vec3{{4, 5, 6}})

		if err := c.FinalizeCurrentMode(); err != nil {
			t.Fatal(err)
		}

		out, _, hasOut := c.PointCloud()
		if !hasOut {
			t.Fatal("PointCloud is not stored")
		}
		expectPointCloud(t, out, []mat.Vec3{{1, 2, 3}, {4, 5, 6}})

		if _, _, hasOutSub1 := c.SubPointCloud(); hasOutSub1 {
			t.Fatal("Sub PointCloud must be cleared")
		}
	})
	t.Run("CancelImportSubPCD", func(t *testing.T) {
		c := newCommandContext(&dummyPCDIO{}, nil)
		if err := c.ImportPCD(pp0); err != nil {
			t.Fatal(err)
		}
		if err := c.ImportSubPCD(pp1); err != nil {
			t.Fatal(err)
		}

		outSub0, _, hasOutSub0 := c.SubPointCloud()
		if !hasOutSub0 {
			t.Fatal("Sub PointCloud is not stored")
		}
		expectPointCloud(t, outSub0, []mat.Vec3{{4, 5, 6}})

		c.UnsetCursors()

		out, _, hasOut := c.PointCloud()
		if !hasOut {
			t.Fatal("PointCloud is not stored")
		}
		expectPointCloud(t, out, []mat.Vec3{{1, 2, 3}})

		if _, _, hasOutSub1 := c.SubPointCloud(); hasOutSub1 {
			t.Fatal("Sub PointCloud must be cleared")
		}
	})
}

func TestExportPCD(t *testing.T) {
	header := pc.PointCloudHeader{
		Fields: []string{"x", "y", "z"},
		Size:   []int{4, 4, 4},
		Type:   []string{"F", "F", "F"},
		Count:  []int{1, 1, 1},
		Width:  4,
		Height: 1,
	}
	pp := &pc.PointCloud{
		PointCloudHeader: header,
		Points:           4,
		Data:             make([]byte, 4*3*4),
	}
	it, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	it.SetVec3(mat.Vec3{1, 2, 3})
	it.Incr()
	it.SetVec3(mat.Vec3{4, 5, 6})
	it.Incr()
	it.SetVec3(mat.Vec3{7, 8, 9})
	it.Incr()
	it.SetVec3(mat.Vec3{10, 11, 12})

	c := newCommandContext(&dummyPCDIO{}, nil)
	c.SetPointCloud(pp, cloudMain)
	c.SetSelectMask([]uint32{
		0,
		selectBitmaskSegmentSelected,
		selectBitmaskSegmentSelected,
		0,
	})
	c.selectMode = selectModeMask

	t.Run("ExportPCD", func(t *testing.T) {
		blob, err := c.ExportPCD()
		if err != nil {
			t.Fatal(err)
		}

		out := blob.(*pc.PointCloud)
		expectPointCloud(t, out, []mat.Vec3{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
			{10, 11, 12},
		})
	})
	t.Run("ExportSelectedPCD", func(t *testing.T) {
		blob, err := c.ExportSelectedPCD()
		if err != nil {
			t.Fatal(err)
		}

		out := blob.(*pc.PointCloud)
		expectPointCloud(t, out, []mat.Vec3{
			{4, 5, 6},
			{7, 8, 9},
		})
	})
}

type dummyPCDIO struct{}

func (dummyPCDIO) importPCD(blob interface{}) (*pc.PointCloud, error) {
	return blob.(*pc.PointCloud), nil
}

func (dummyPCDIO) exportPCD(pp *pc.PointCloud) (interface{}, error) {
	return pp, nil
}

func TestBaseFileter(t *testing.T) {
	c := &commandContext{
		selectMask: []uint32{
			0,
			selectBitmaskCropped | selectBitmaskSelected,
			selectBitmaskSelected,
			selectBitmaskSegmentSelected,
			selectBitmaskCropped | selectBitmaskSegmentSelected,
		},
	}
	check := func(t *testing.T, expected map[int]bool, f func(int, mat.Vec3) bool) {
		t.Helper()
		for id, val := range expected {
			if out := f(id, mat.Vec3{}); out != val {
				t.Errorf("%d is expected to be %v, got %v", id, val, out)
			}
		}
	}
	t.Run("ExtractSelected", func(t *testing.T) {
		check(t, map[int]bool{
			0: false,
			1: false,
			2: true,
			3: false,
			4: false,
		}, c.baseFilter(true))
	})
	t.Run("ExtractNotSelected", func(t *testing.T) {
		check(t, map[int]bool{
			0: true,
			1: true,
			2: false,
			3: true,
			4: true,
		}, c.baseFilter(false))
	})
	t.Run("ExtractSegmentSelected", func(t *testing.T) {
		check(t, map[int]bool{
			0: false,
			1: false,
			2: false,
			3: true,
			4: false,
		}, c.baseFilterByMask(true))
	})
	t.Run("ExtractSegmentNotSelected", func(t *testing.T) {
		check(t, map[int]bool{
			0: true,
			1: true,
			2: true,
			3: false,
			4: true,
		}, c.baseFilterByMask(false))
	})
}
