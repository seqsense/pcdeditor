package main

import (
	"errors"
	"fmt"
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
	errSetCursor      = errors.New("failed to set cursor")
)

var consoleCommands = map[string]func(c *console, sel []uint32, args []float32) ([][]float32, error){
	"mem": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		var stat runtime.MemStats
		runtime.ReadMemStats(&stat)
		fmt.Printf("%+v\n", stat)
		return nil, nil
	},
	"select_range": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.SelectRange()}}, nil
		case 1:
			c.cmd.SetSelectRange(args[0])
			return [][]float32{{c.cmd.SelectRange()}}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"cursor": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"unset_cursor": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.UnsetCursors()
		return nil, nil
	},
	"snap_v": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SnapVertical()
		return nil, nil
	},
	"snap_h": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SnapHorizontal()
		return nil, nil
	},
	"translate_cursor": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 3 {
			return nil, errArgumentNumber
		}
		c.cmd.TransformCursors(mat.Translate(args[0], args[1], args[2]))
		return nil, nil
	},
	"add_surface": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"delete": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.Delete(sel)
		return nil, nil
	},
	"label": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		c.cmd.Label(sel, uint32(args[0]))
		return nil, nil
	},
	"undo": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.Undo()
		return nil, nil
	},
	"max_history": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"crop": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.Crop()
		return nil, nil
	},
	"map_alpha": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"point_size": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{c.cmd.PointSize()}}, nil
		case 1:
			return nil, c.cmd.SetPointSize(args[0])
		default:
			return nil, errArgumentNumber
		}
	},
	"fov": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"voxel_grid": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{}, c.cmd.VoxelFilter(sel, defaultResolution)
		case 1:
			return [][]float32{}, c.cmd.VoxelFilter(sel, args[0])
		default:
			return nil, errArgumentNumber
		}
	},
	"z_range": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"ortho": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SetProjectionType(ProjectionOrthographic)
		return nil, nil
	},
	"perspective": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.cmd.SetProjectionType(ProjectionPerspective)
		return nil, nil
	},
	"rotate_yaw": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		c.view.RotateYaw(float64(args[0]))
		return nil, nil
	},
	"pitch": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		c.view.SetPitch(float64(args[0]))
		return nil, nil
	},
	"snap_pitch": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.SnapPitch()
		return nil, nil
	},
	"snap_yaw": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.SnapYaw()
		return nil, nil
	},
	"segmentation_param": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
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
	"view_reset": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.Reset()
		return nil, nil
	},
	"view_fps": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		c.view.FPS()
		return nil, nil
	},
	"view": func(c *console, sel []uint32, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			params := c.view.View()
			ret := make([]float32, 5)
			for i, v := range params {
				ret[i] = float32(v)
			}
			return [][]float32{ret}, nil
		case 5:
			var params [5]float64
			for i, v := range args {
				params[i] = float64(v)
			}
			return nil, c.view.SetView(params)
		default:
			return nil, errArgumentNumber
		}
	},
}

func (c *console) Run(line string, sel []uint32) ([][]float32, error) {
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
	res, err := fn(c, sel, argsFloat)
	if err != nil {
		return nil, err
	}
	return res, nil
}
