package main

import (
	"testing"
)

func TestConsole_SelectRange(t *testing.T) {
	c := &console{
		cmd: newCommandContext(nil, nil),
	}
	c.cmd.SetProjectionType(ProjectionPerspective)

	c.Run("select_range 123", nil)
	if v := c.cmd.SelectRange(rangeTypeAuto); v != 123 {
		t.Errorf("SelectRangeAuto must be updated, expected: 123, got: %f", v)
	}

	c.Run("select_range_perspective 124", nil)
	if v := c.cmd.SelectRange(rangeTypeAuto); v != 124 {
		t.Errorf("SelectRangeAuto must be updated by setting rangeTypePerspective, expected: 124, got: %f", v)
	}
	if v := c.cmd.SelectRange(rangeTypePerspective); v != 124 {
		t.Errorf("SelectRangePerspective must be updated, expected: 124, got: %f", v)
	}

	c.Run("select_range_ortho 125", nil)
	if v := c.cmd.SelectRange(rangeTypeAuto); v != 124 {
		t.Errorf("SelectRangeAuto must not be updated by setting rangeTypeOrtho, expected: 124, got: %f", v)
	}
	if v := c.cmd.SelectRange(rangeTypeOrtho); v != 125 {
		t.Errorf("SelectRangeOrtho must be updated, expected: 125, got: %f", v)
	}

	c.Run("ortho", nil)
	if v := c.cmd.SelectRange(rangeTypeAuto); v != 125 {
		t.Errorf("SelectRangeAuto must not be updated by setting rangeTypeOrtho, expected: 125, got: %f", v)
	}
}

func TestConsole_ScanCache(t *testing.T) {
	c := &console{
		cmd: newCommandContext(nil, nil),
	}

	res, err := c.Run("scan_cache", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res[0][0] != 1 {
		t.Errorf("scan_cache must be enabled by default, got: %f", res[0][0])
	}

	if _, err := c.Run("scan_cache 0", nil); err != nil {
		t.Fatal(err)
	}
	if c.cmd.ScanCacheEnabled() {
		t.Error("scan_cache 0 must disable the scan cache")
	}
	res, err = c.Run("scan_cache", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res[0][0] != 0 {
		t.Errorf("scan_cache must report the disabled state, got: %f", res[0][0])
	}

	if _, err := c.Run("scan_cache 1", nil); err != nil {
		t.Fatal(err)
	}
	if !c.cmd.ScanCacheEnabled() {
		t.Error("scan_cache 1 must enable the scan cache")
	}
}
