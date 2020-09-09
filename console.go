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

var errArgumentNumber = errors.New("invalid number of arguments")
var errInvalidCommand = errors.New("invalid command")

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
	"cursors": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			var resFloat [][]float32
			for i, c := range cmd.Cursors() {
				resFloat = append(resFloat, []float32{float32(i), c[0], c[1], c[2]})
			}
			return resFloat, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"cursor": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 3:
			n := len(cmd.Cursors())
			if !cmd.SetCursor(n, mat.Vec3{args[0], args[1], args[2]}) {
				return nil, errors.New("failed to set cursor")
			}
			return [][]float32{{float32(n), args[0], args[1], args[2]}}, nil
		case 4:
			if !cmd.SetCursor(int(args[0]), mat.Vec3{args[1], args[2], args[3]}) {
				return nil, errors.New("failed to set cursor")
			}
			return [][]float32{args}, nil
		default:
			return nil, errArgumentNumber
		}
	},
	"unset_cursors": func(cmd *commandContext, args []float32) ([][]float32, error) {
		switch len(args) {
		case 0:
			cmd.UnsetCursors()
			return nil, nil
		default:
			return nil, errArgumentNumber
		}
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
