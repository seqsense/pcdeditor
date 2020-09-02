package main

import (
	"bytes"
	"errors"
	"fmt"
	"syscall/js"
	"time"

	webgl "github.com/seqsense/pcdviewer/gl"
	"github.com/seqsense/pcdviewer/mat"
	"github.com/seqsense/pcdviewer/pcd"
)

func main() {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "mapCanvas")

	logDiv := doc.Call("getElementById", "log")
	logPrint := func(msg interface{}) {
		html := logDiv.Get("innerHTML").String()
		logDiv.Set("innerHTML", fmt.Sprintf("%s%s<br/>", html, msg))
	}

	gl, err := webgl.New(canvas)
	if err != nil {
		logPrint(err)
		return
	}

	vs := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vs, vsSource)
	gl.CompileShader(vs)
	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(fs, fsSource)
	gl.CompileShader(fs)

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	projectionMatrixLocation := gl.GetUniformLocation(program, "uProjectionMatrix")
	modelViewMatrixLocation := gl.GetUniformLocation(program, "uModelViewMatrix")

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	gl.UseProgram(program)

	updateProjectionMatrix := func(width, height int) {
		gl.Canvas.SetWidth(width)
		gl.Canvas.SetHeight(height)
		projectionMatrix := mat.Perspective(
			45*3.14/180,
			float32(width)/float32(height),
			1.0, 1000.0,
		)
		gl.UniformMatrix4fv(projectionMatrixLocation, false, projectionMatrix)
		gl.Viewport(0, 0, width, height)
	}
	width := gl.Canvas.ClientWidth()
	height := gl.Canvas.ClientHeight()
	updateProjectionMatrix(width, height)

	ang := float32(0.0)
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()

	chNewPath := make(chan string)
	js.Global().Set("loadPCD",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chNewPath <- args[0].String()
			return nil
		}),
	)

	chUpdateView := make(chan float32)
	viewDistance := float32(100.0)
	var modelViewMatrixBase mat.Mat4
	updateView := func(distance float32) {
		modelViewMatrixBase =
			mat.Translate(0, 0, -distance).
				Mul(mat.Rotate(1, 0, 0, 3.14/4))
	}
	updateView(viewDistance)

	canvas.Call("addEventListener", "wheel",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := args[0]
			event.Call("preventDefault")
			viewDistance += float32(event.Get("deltaY").Int())
			chUpdateView <- viewDistance
			return false
		}),
	)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	var nPoints int
	for {
		newWidth := gl.Canvas.ClientWidth()
		newHeight := gl.Canvas.ClientHeight()
		if newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			updateProjectionMatrix(width, height)
		}

		modelViewMatrix := modelViewMatrixBase.Mul(mat.Rotate(0, 0, 1, ang))
		gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if nPoints > 0 {
			gl.DrawArrays(gl.POINTS, 0, nPoints)
		}

		for {
			select {
			case path := <-chNewPath:
				logPrint("loading pcd file")
				n, err := loadPCD(gl, program, path)
				if err != nil {
					logPrint(err)
					continue
				}
				logPrint("pcd file loaded")
				nPoints = n
				continue
			case d := <-chUpdateView:
				updateView(d)
				continue
			case <-tick.C:
			}
			break
		}

		ang += 0.01
	}
}

func loadPCD(gl *webgl.WebGL, program webgl.Program, path string) (int, error) {
	var b []byte
	chErr := make(chan error)
	js.Global().Call("fetch", path).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].Get("ok").Bool() {
				args[0].Call("arrayBuffer").Call("then",
					js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						array := js.Global().Get("Uint8Array").New(args[0])
						n := array.Get("byteLength").Int()
						b = make([]byte, n)
						js.CopyBytesToGo(b, array)
						chErr <- nil
						return nil
					}),
				)
				return nil
			}
			chErr <- errors.New("failed to fetch file")
			return nil
		}),
	)

	if err := <-chErr; err != nil {
		return 0, err
	}

	pc, err := pcd.Parse(bytes.NewReader(b))
	if err != nil {
		return 0, err
	}

	vertexPosition := gl.GetAttribLocation(program, "aVertexPosition")
	posBuf := gl.CreateBuffer()

	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.VertexAttribPointer(vertexPosition, 3, gl.FLOAT, false, 0, pc.Stride())
	gl.EnableVertexAttribArray(vertexPosition)

	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)

	return pc.Points - 1, nil
}

const vsSource = `
	attribute vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	varying lowp vec4 vColor;
	const float zMax = 5.0;
	const float zMin = -5.0;
	const float zRange = zMax - zMin;
	varying lowp float c;
	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 2.0;

		if (aVertexPosition[2] < zMin) {
			c = 0.0;
		} else if (aVertexPosition[2] > zMax) {
			c = 1.0;
		} else {
			c = (aVertexPosition[2] - zMin) / zRange;
		}
		vColor = vec4(c, 0.0, 1.0 - c, 1.0);
	}
`
const fsSource = `
	varying lowp vec4 vColor;
	void main(void) {
		gl_FragColor = vColor;
	}
`
