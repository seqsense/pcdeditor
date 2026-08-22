package main

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
)

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
		e.merge(makeTestCloud(t, n, n, 1))
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

func TestHistoryMaxDepth(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))

	e := newEditor() // maxHistoryDefault = 4
	if err := e.SetPointCloud(makeTestCloud(t, 100, 10, 10), cloudMain); err != nil {
		t.Fatal(err)
	}

	for k := 0; k < 6; k++ {
		applyRandomEdit(t, e, rnd)
	}
	for k := 0; k < maxHistoryDefault; k++ {
		if !e.Undo() {
			t.Fatalf("undo %d must succeed", k)
		}
	}
	if e.Undo() {
		t.Fatal("undo deeper than max_history must fail")
	}

	e.SetMaxHistory(0)
	applyRandomEdit(t, e, rnd)
	if e.Undo() {
		t.Fatal("undo with max_history=0 must fail")
	}
}

func TestHistorySquashLatest(t *testing.T) {
	e := newEditor()
	if err := e.SetPointCloud(makeTestCloud(t, 100, 10, 10), cloudMain); err != nil {
		t.Fatal(err)
	}
	orig := snapshotCloud(e)

	if err := e.passThrough(func(i int, _ mat.Vec3) bool { return i%2 == 0 }); err != nil {
		t.Fatal(err)
	}
	e.merge(makeTestCloud(t, 10, 10, 1))
	e.squashLatest()

	if !e.Undo() {
		t.Fatal("undo failed")
	}
	assertCloudEqual(t, orig, e.pp)
}

func TestHistoryUndoKeepsEntryOnError(t *testing.T) {
	h := &historyMem{
		maxHistory: 4,
		entries:    [][][]byte{{{0xFF}}}, // broken patch data
	}
	if _, ok := h.undo(nil); ok {
		t.Fatal("undo of a broken entry must fail")
	}
	if len(h.entries) != 1 {
		t.Fatal("a broken entry must not be dropped; a later undo would apply an older patch to a mismatched state")
	}
}
