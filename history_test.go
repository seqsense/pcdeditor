package main

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

func makeCloud(t *testing.T, points []mat.Vec3, labels []uint32) *pc.PointCloud {
	t.Helper()
	pp := &pc.PointCloud{
		PointCloudHeader: pc.PointCloudHeader{
			Version:   0.7,
			Fields:    []string{"x", "y", "z", "label"},
			Size:      []int{4, 4, 4, 4},
			Type:      []string{"F", "F", "F", "U"},
			Count:     []int{1, 1, 1, 1},
			Width:     len(points),
			Height:    1,
			Viewpoint: []float32{0, 0, 0, 1, 0, 0, 0},
		},
		Points: len(points),
	}
	pp.Data = make([]byte, len(points)*pp.Stride())
	it, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	lt, err := pp.Uint32Iterator("label")
	if err != nil {
		t.Fatal(err)
	}
	for i := range points {
		it.SetVec3(points[i])
		lt.SetUint32(labels[i])
		it.Incr()
		lt.Incr()
	}
	return pp
}

func assertCloud(t *testing.T, pp *pc.PointCloud, points []mat.Vec3, labels []uint32) {
	t.Helper()
	if pp.Points != len(points) {
		t.Fatalf("Expected %d points, got %d", len(points), pp.Points)
	}
	it, err := pp.Vec3Iterator()
	if err != nil {
		t.Fatal(err)
	}
	lt, err := pp.Uint32Iterator("label")
	if err != nil {
		t.Fatal(err)
	}
	for i := range points {
		if !points[i].Equal(it.Vec3()) {
			t.Fatalf("Point %d: expected %v, got %v", i, points[i], it.Vec3())
		}
		if labels[i] != lt.Uint32() {
			t.Fatalf("Label %d: expected %d, got %d", i, labels[i], lt.Uint32())
		}
		it.Incr()
		lt.Incr()
	}
}

func TestEditorUndo(t *testing.T) {
	pts := []mat.Vec3{{1, 0, 0}, {2, 0, 0}, {3, 0, 0}}

	newTestEditor := func(t *testing.T, labels []uint32) *editor {
		t.Helper()
		e := newEditor()
		if err := e.SetPointCloud(makeCloud(t, pts, labels), cloudMain); err != nil {
			t.Fatal(err)
		}
		return e
	}
	undo := func(t *testing.T, e *editor) {
		t.Helper()
		if !e.Undo() {
			t.Fatal("undo failed")
		}
	}

	t.Run("Label", func(t *testing.T) {
		e := newTestEditor(t, []uint32{0, 0, 0})
		if err := e.label(func(i int, _ mat.Vec3) (uint32, bool) {
			return 5, i != 1
		}); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, pts, []uint32{5, 0, 5})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{0, 0, 0})
	})
	t.Run("Delete", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		if err := e.passThrough(func(i int, _ mat.Vec3) bool {
			return i != 1
		}); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, []mat.Vec3{pts[0], pts[2]}, []uint32{1, 3})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("DeleteByMask", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		sel := []uint32{0, selectBitmaskSegmentSelected, 0}
		if err := e.passThroughByMask(sel, selectBitmaskSegmentSelected, 0); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, []mat.Vec3{pts[0], pts[2]}, []uint32{1, 3})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("Merge", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		if err := e.merge(makeCloud(t, []mat.Vec3{{4, 0, 0}}, []uint32{7})); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, append(pts, mat.Vec3{4, 0, 0}), []uint32{1, 2, 3, 7})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("Relabel", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		if err := e.relabelPointsInLabelRange(1, 2, 9); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, pts, []uint32{9, 9, 3})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("Unlabel", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		if err := e.unlabelPoints([]uint32{2}); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, pts, []uint32{0, 2, 0})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("Replace", func(t *testing.T) {
		e := newTestEditor(t, []uint32{1, 2, 3})
		pts2 := []mat.Vec3{{9, 9, 9}}
		if err := e.SetPointCloud(makeCloud(t, pts2, []uint32{4}), cloudMain); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, pts2, []uint32{4})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 2, 3})
	})
	t.Run("SequentialEdits", func(t *testing.T) {
		e := newTestEditor(t, []uint32{0, 0, 0})
		if err := e.relabelPointsInLabelRange(0, 0, 1); err != nil {
			t.Fatal(err)
		}
		if err := e.passThrough(func(i int, _ mat.Vec3) bool {
			return i == 0
		}); err != nil {
			t.Fatal(err)
		}
		assertCloud(t, e.pp, pts[:1], []uint32{1})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{1, 1, 1})
		undo(t, e)
		assertCloud(t, e.pp, pts, []uint32{0, 0, 0})
	})
	t.Run("NothingToUndo", func(t *testing.T) {
		e := newTestEditor(t, []uint32{0, 0, 0})
		if e.Undo() {
			t.Fatal("undo without edits must fail")
		}
	})
}

func TestHistoryMaxDepth(t *testing.T) {
	pts := []mat.Vec3{{1, 0, 0}, {2, 0, 0}}
	labelAll := func(t *testing.T, e *editor, l uint32) {
		t.Helper()
		if err := e.label(func(int, mat.Vec3) (uint32, bool) {
			return l, true
		}); err != nil {
			t.Fatal(err)
		}
	}

	e := newEditor()
	e.SetMaxHistory(2)
	if err := e.SetPointCloud(makeCloud(t, pts, []uint32{0, 0}), cloudMain); err != nil {
		t.Fatal(err)
	}
	labelAll(t, e, 1)
	labelAll(t, e, 2)
	labelAll(t, e, 3)

	if !e.Undo() {
		t.Fatal("first undo must succeed")
	}
	assertCloud(t, e.pp, pts, []uint32{2, 2})
	if !e.Undo() {
		t.Fatal("second undo must succeed")
	}
	assertCloud(t, e.pp, pts, []uint32{1, 1})
	if e.Undo() {
		t.Fatal("undo deeper than max_history must fail")
	}
	assertCloud(t, e.pp, pts, []uint32{1, 1})

	e.SetMaxHistory(0)
	labelAll(t, e, 9)
	if e.Undo() {
		t.Fatal("undo with max_history=0 must fail")
	}
}

func TestHistorySquashLatest(t *testing.T) {
	pts := []mat.Vec3{{1, 0, 0}, {2, 0, 0}}

	e := newEditor()
	if err := e.SetPointCloud(makeCloud(t, pts, []uint32{0, 0}), cloudMain); err != nil {
		t.Fatal(err)
	}
	if err := e.relabelPointsInLabelRange(0, 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := e.merge(makeCloud(t, []mat.Vec3{{3, 0, 0}}, []uint32{2})); err != nil {
		t.Fatal(err)
	}
	e.squashLatest()

	if !e.Undo() {
		t.Fatal("undo failed")
	}
	assertCloud(t, e.pp, pts, []uint32{0, 0})
	if e.Undo() {
		t.Fatal("squashed edits must be undone as a single step")
	}
}

// The tests below fuzz the same behavior with randomized edit sequences.

func snapshotCloud(e *editor) *pc.PointCloud {
	return cloneCloud(e.pp)
}

func applyRandomEdit(t *testing.T, e *editor, rnd *rand.Rand) {
	t.Helper()
	switch rnd.Intn(5) {
	case 0: // label by position
		if err := e.label(func(i int, _ mat.Vec3) (uint32, bool) {
			return uint32(rnd.Intn(4)), rnd.Intn(2) == 0
		}); err != nil {
			t.Fatal(err)
		}
	case 1: // relabel range
		if err := e.relabelPointsInLabelRange(0, uint32(rnd.Intn(3)), uint32(rnd.Intn(4))); err != nil {
			t.Fatal(err)
		}
	case 2: // delete random points
		if err := e.passThrough(func(i int, _ mat.Vec3) bool {
			return rnd.Intn(4) != 0
		}); err != nil {
			t.Fatal(err)
		}
	case 3: // paste
		n := 1 + rnd.Intn(20)
		if err := e.merge(makeTestCloud(t, n, n, 1)); err != nil {
			t.Fatal(err)
		}
	case 4: // whole-cloud replacement
		n := 50 + rnd.Intn(100)
		if err := e.SetPointCloud(makeTestCloud(t, n, n, 1), cloudMain); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEditorUndoRoundTrip(t *testing.T) {
	for trial := int64(0); trial < 10; trial++ {
		rnd := rand.New(rand.NewSource(trial))

		e := newEditor()
		e.SetMaxHistory(100)
		if err := e.SetPointCloud(makeTestCloud(t, 200, 20, 10), cloudMain); err != nil {
			t.Fatal(err)
		}

		const nOps = 8
		snapshots := []*pc.PointCloud{snapshotCloud(e)}
		for k := 0; k < nOps; k++ {
			applyRandomEdit(t, e, rnd)
			snapshots = append(snapshots, snapshotCloud(e))
		}

		for k := nOps; k > 0; k-- {
			assertCloudEqual(t, snapshots[k], e.pp)
			if !e.Undo() {
				t.Fatalf("trial %d: undo %d failed", trial, nOps-k)
			}
		}
		assertCloudEqual(t, snapshots[0], e.pp)
		if !reflect.DeepEqual(snapshots[0].PointCloudHeader, e.pp.PointCloudHeader) {
			t.Fatalf("trial %d: header mismatch after undoing all edits", trial)
		}

		if e.Undo() {
			t.Fatalf("trial %d: undo over the initial state must fail", trial)
		}
	}
}
