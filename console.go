package main

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/seqsense/pcgol/mat"
)

type console struct {
	cmd  *commandContext
	view view
}

var (
	errArgumentNumber = errors.New("invalid number of arguments")
	errInvalidCommand = errors.New("invalid command")
	errOutOfRange     = errors.New("out of range")
	errSetCursor      = errors.New("failed to set cursor")
)

type updateSelectionFn func() error

var consoleCommands = map[string]func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error){
	"mem": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		var stat runtime.MemStats
		runtime.ReadMemStats(&stat)
		fmt.Printf("%+v\n", stat)
		return nil, nil
	},
	"select_range": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.SelectRange(rangeTypeAuto)}}, nil
		case 1:
			c.cmd.SetSelectRange(rangeTypeAuto, args[0])
			return [][]float32{{c.cmd.SelectRange(rangeTypeAuto)}}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"select_range_perspective": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.SelectRange(rangeTypePerspective)}}, nil
		case 1:
			c.cmd.SetSelectRange(rangeTypePerspective, args[0])
			return [][]float32{{c.cmd.SelectRange(rangeTypePerspective)}}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"select_range_ortho": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.SelectRange(rangeTypeOrtho)}}, nil
		case 1:
			c.cmd.SetSelectRange(rangeTypeOrtho, args[0])
			return [][]float32{{c.cmd.SelectRange(rangeTypeOrtho)}}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"cursor": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			var resFloat [][]float32
			for i, c := range c.cmd.Cursors() {
				resFloat = append(resFloat, []float32{float32(i), c[0], c[1], c[2]})
			}
			return resFloat, nil
		case 3:
			n := len(c.cmd.Cursors())
			if !c.cmd.SetCursor(n, mat.Vec3{args[0], args[1], args[2]}) {
				return nil, errSetCursor
			}
			return [][]float32{{float32(n), args[0], args[1], args[2]}}, nil
		case 4:
			if !c.cmd.SetCursor(int(args[0]), mat.Vec3{args[1], args[2], args[3]}) {
				return nil, errSetCursor
			}
			return [][]float32{args}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"unset_cursor": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.UnsetCursors()
		return nil, nil
	},
	"snap_v": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SnapVertical()
		return nil, nil
	},
	"snap_h": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SnapHorizontal()
		return nil, nil
	},
	"translate_cursor": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 3 {
			return nil, errArgumentNumber
		}
		c.cmd.TransformCursors(mat.Translate(args[0], args[1], args[2]))
		return nil, nil
	},
	"add_surface": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			c.cmd.AddSurface(defaultResolution)
			return nil, nil
		case 1:
			c.cmd.AddSurface(args[0])
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"delete": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		if err := updateSel(); err != nil {
			return nil, err
		}
		c.cmd.Delete()
		return nil, nil
	},
	"label": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		if err := updateSel(); err != nil {
			return nil, err
		}
		c.cmd.Label(uint32(args[0]))
		return nil, nil
	},
	"undo": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.Undo()
		return nil, nil
	},
	"max_history": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{float32(c.cmd.MaxHistory())}}, nil
		case 1:
			c.cmd.SetMaxHistory(int(args[0]))
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"crop": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.Crop()
		return nil, nil
	},
	"map_alpha": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.MapAlpha()}}, nil
		case 1:
			c.cmd.SetMapAlpha(args[0])
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"point_size": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.PointSize()}}, nil
		case 1:
			return nil, c.cmd.SetPointSize(args[0])
		default:
			return nil, errArgumentNumber
		}
	},
	"fov": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 1:
			switch {
			case args[0] > 0:
				c.view.IncreaseFOV()
			case args[0] < 0:
				c.view.DecreaseFOV()
			}
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"voxel_grid": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if err := updateSel(); err != nil {
			return nil, err
		}
		switch len(args) {
		case 0:
			return [][]float32{}, c.cmd.VoxelFilter(defaultResolution)
		case 1:
			return [][]float32{}, c.cmd.VoxelFilter(args[0])
		default:
			return nil, errArgumentNumber
		}
	},
	"z_range": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			zMin, zMax := c.cmd.ZRange()
			return [][]float32{{zMin, zMax}}, nil
		case 2:
			c.cmd.SetZRange(args[0], args[1])
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"ortho": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SetProjectionType(ProjectionOrthographic)
		return nil, nil
	},
	"perspective": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SetProjectionType(ProjectionPerspective)
		return nil, nil
	},
	"rotate_yaw": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		c.view.RotateYaw(float64(args[0]))
		return nil, nil
	},
	"pitch": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		c.view.SetPitch(float64(args[0]))
		return nil, nil
	},
	"snap_pitch": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.SnapPitch()
		return nil, nil
	},
	"snap_yaw": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.SnapYaw()
		return nil, nil
	},
	"segmentation_param": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			p0, p1 := c.cmd.SegmentationParam()
			return [][]float32{{p0, p1}}, nil
		case 2:
			return nil, c.cmd.SetSegmentationParam(args[0], args[1])
		default:
			return nil, errArgumentNumber
		}
	},
	"view_reset": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.Reset()
		return nil, nil
	},
	"view_fps": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.FPS()
		return nil, nil
	},
	"view": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			x, y, yaw, pitch, distance := c.view.View()
			return [][]float32{{
				float32(x), float32(y),
				float32(yaw), float32(pitch), float32(distance),
			}}, nil
		case 5:
			x, y, yaw, pitch, distance :=
				float64(args[0]), float64(args[1]),
				float64(args[2]), float64(args[3]), float64(args[4])
			return nil, c.view.SetView(x, y, yaw, pitch, distance)
		default:
			return nil, errArgumentNumber
		}
	},
	"fit_inserting": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		var axes [6]bool
		for _, v := range args {
			i := int(math.Round(float64(v)))
			if i < 0 || 6 <= i {
				return nil, errOutOfRange
			}
			axes[i] = true
		}
		return nil, c.cmd.FitInserting(axes)
	},
	"label_segmentation_param": func(c *console, updateSel updateSelectionFn, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			p := c.cmd.LabelSegmentationParam()
			return [][]float32{{p}}, nil
		case 2:
			return nil, c.cmd.SetLabelSegmentationParam(args[0])
		default:
			return nil, errArgumentNumber
		}
	},
}

func (c *console) Run(line string, updateSel updateSelectionFn) ([][]float32, error) {
	args := strings.Fields(line)
	if len(args) == 0 {
		return nil, nil
	}
	fn, ok := consoleCommands[args[0]]
	if !ok {
		return nil, errInvalidCommand
	}
	var argsFloat []float32
	for i := 1; i < len(args); i++ {
		f, err := strconv.ParseFloat(args[i], 32)
		if err != nil {
			return nil, err
		}
		argsFloat = append(argsFloat, float32(f))
	}
	res, err := fn(c, updateSel, argsFloat)
	if err != nil {
		return nil, err
	}
	return res, nil
}
