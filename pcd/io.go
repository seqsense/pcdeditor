package pcd

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/zhuyie/golzf"
)

type Format int

const (
	Ascii Format = iota
	Binary
	BinaryCompressed
)

func Parse(r io.Reader) (*PointCloud, error) {
	rb := bufio.NewReader(r)
	pc := &PointCloud{}
	var fmt Format

L_HEADER:
	for {
		line, _, err := rb.ReadLine()
		if err != nil {
			return nil, err
		}
		args := strings.Fields(string(line))
		if len(args) < 2 {
			return nil, errors.New("header field must have value")
		}
		switch args[0] {
		case "VERSION":
			f, err := strconv.ParseFloat(args[1], 32)
			if err != nil {
				return nil, err
			}
			pc.Version = float32(f)
		case "FIELDS":
			pc.Fields = args[1:]
		case "SIZE":
			pc.Size = make([]int, len(args)-1)
			for i, s := range args[1:] {
				pc.Size[i], err = strconv.Atoi(s)
				if err != nil {
					return nil, err
				}
			}
		case "TYPE":
			pc.Type = args[1:]
		case "COUNT":
			pc.Count = make([]int, len(args)-1)
			for i, s := range args[1:] {
				pc.Count[i], err = strconv.Atoi(s)
				if err != nil {
					return nil, err
				}
			}
		case "WIDTH":
			pc.Width, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, err
			}
		case "HEIGHT":
			pc.Height, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, err
			}
		case "VIEWPOINT":
			pc.Viewpoint = make([]float32, len(args)-1)
			for i, s := range args[1:] {
				f, err := strconv.ParseFloat(s, 32)
				if err != nil {
					return nil, err
				}
				pc.Viewpoint[i] = float32(f)
			}
		case "POINTS":
			pc.Points, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, err
			}
		case "DATA":
			switch args[1] {
			case "ascii":
				fmt = Ascii
			case "binary":
				fmt = Binary
			case "binary_compressed":
				fmt = BinaryCompressed
			default:
				return nil, errors.New("unknown data format")
			}
			break L_HEADER
		}
	}
	// validate
	if len(pc.Fields) != len(pc.Size) {
		return nil, errors.New("size field size is wrong")
	}
	if len(pc.Fields) != len(pc.Type) {
		return nil, errors.New("type field size is wrong")
	}
	if len(pc.Fields) != len(pc.Count) {
		return nil, errors.New("count field size is wrong")
	}

	switch fmt {
	case Ascii:
		panic("not implemented yet")
	case Binary:
		b, err := ioutil.ReadAll(rb)
		if err != nil {
			return nil, err
		}
		pc.Data = b
	case BinaryCompressed:
		var nCompressed, nUncompressed int32
		if err := binary.Read(rb, binary.LittleEndian, &nCompressed); err != nil {
			return nil, err
		}
		if err := binary.Read(rb, binary.LittleEndian, &nUncompressed); err != nil {
			return nil, err
		}

		b, err := ioutil.ReadAll(rb)
		if err != nil {
			return nil, err
		}

		dec := make([]byte, nUncompressed)
		n, err := lzf.Decompress(b[:nCompressed], dec)
		if err != nil {
			return nil, err
		}
		if int(nUncompressed) != n {
			return nil, errors.New("wrong uncompressed size")
		}

		head := make([]int, len(pc.Fields))
		offset := make([]int, len(pc.Fields))
		var pos, off int
		for i := range pc.Fields {
			head[i] = pos
			offset[i] = off
			pos += pc.Size[i] * pc.Count[i] * pc.Points
			off += pc.Size[i] * pc.Count[i]
		}

		stride := pc.Stride()
		pc.Data = make([]byte, n)
		for p := 0; p < pc.Points; p++ {
			for i := range head {
				size := pc.Size[i]
				to := p*stride + offset[i]
				from := head[i] + p*size
				copy(pc.Data[to:to+size], dec[from:from+size])
			}
		}
	}

	return pc, nil
}
