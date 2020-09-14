package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/seqsense/pcdeditor/mat"
)

type console struct {
	cmd *commandContext
}

var (
	errArgumentNumber = errors.New("invalid number of arguments")
	errInvalidCommand = errors.New("invalid command")
	errSetCursor      = errors.New("failed to set cursor")
)

var consoleCommands = map[string]func(cmd *commandContext, args []float32) ([][]float32, error){
	"select_range": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			return [][]float32{{cmd.SelectRange()}}, nil
		case 1:
			cmd.SetSelectRange(args[0])
			return [][]float32{{cmd.SelectRange()}}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"cursor": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			var resFloat [][]float32
			for i, c := range cmd.Cursors() {
				resFloat = append(resFloat, []float32{float32(i), c[0], c[1], c[2]})
			}
			return resFloat, nil
		case 3:
			n := len(cmd.Cursors())
			if !cmd.SetCursor(n, mat.Vec3{args[0], args[1], args[2]}) {
				return nil, errSetCursor
			}
			return [][]float32{{float32(n), args[0], args[1], args[2]}}, nil
		case 4:
			if !cmd.SetCursor(int(args[0]), mat.Vec3{args[1], args[2], args[3]}) {
				return nil, errSetCursor
			}
			return [][]float32{args}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"unset_cursor": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.UnsetCursors()
		return nil, nil
	},
	"snap_v": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.SnapVertical()
		return nil, nil
	},
	"snap_h": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.SnapHorizontal()
		return nil, nil
	},
	"translate_cursor": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 3 {
			return nil, errArgumentNumber
		}
		cmd.TransformCursors(mat.Translate(args[0], args[1], args[2]))
		return nil, nil
	},
	"add_surface": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			cmd.AddSurface(defaultResolution)
			return nil, nil
		case 1:
			cmd.AddSurface(args[0])
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"delete": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.Delete()
		return nil, nil
	},
	"label": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		cmd.Label(uint32(args[0]))
		return nil, nil
	},
	"undo": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.Undo()
		return nil, nil
	},
	"crop": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 0 {
			return nil, errArgumentNumber
		}
		cmd.Crop()
		return nil, nil
	},
	"map_alpha": func(cmd *commandContext, args []float32) ([][]float32, error) {
		if len(args) != 1 {
			return nil, errArgumentNumber
		}
		cmd.SetMapAlpha(args[0])
		return nil, nil
	},
}

func (c *console) Run(line string) (string, error) {
	args := strings.Fields(line)
	if len(args) == 0 {
		return "", nil
	}
	fn, ok := consoleCommands[args[0]]
	if !ok {
		return "", errInvalidCommand
	}
	var argsFloat []float32
	for i := 1; i < len(args); i++ {
		f, err := strconv.ParseFloat(args[i], 32)
		if err != nil {
			return "", err
		}
		argsFloat = append(argsFloat, float32(f))
	}
	res, err := fn(c.cmd, argsFloat)
	if err != nil {
		return "", err
	}
	var resStr []string
	for _, vv := range res {
		var resLine []string
		for _, v := range vv {
			resLine = append(resLine, strconv.FormatFloat(float64(v), 'f', 3, 32))
		}
		resStr = append(resStr, strings.Join(resLine, " "))
	}
	return strings.Join(resStr, "\n"), nil
}
