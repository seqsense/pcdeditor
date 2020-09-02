package main

import (
	//"math"
	"net/http"
	"syscall/js"
	"time"

	webgl "github.com/seqsense/pcdviewer/gl"
	"github.com/seqsense/pcdviewer/mat"
	"github.com/seqsense/pcdviewer/pcd"
)

func main() {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "mapCanvas")
	gl, err := webgl.New(canvas)
	if err != nil {
		panic(err)
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

	chNewURL := make(chan string)
	js.Global().Set("loadPCD",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chNewURL <- args[0].String()
			return nil
		}),
	)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	var nPoints int
	modelViewMatrixBase := mat.Translate(0, 0, -100).Mul(mat.Rotate(1, 0, 0, 3.14/4))
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
			case url := <-chNewURL:
				nPoints = loadPCD(gl, program, url)
				continue
			case <-tick.C:
			}
			break
		}

		ang += 0.01
	}
}

func loadPCD(gl *webgl.WebGL, program webgl.Program, url string) int {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	pc, err := pcd.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}

	vertexPosition := gl.GetAttribLocation(program, "aVertexPosition")
	posBuf := gl.CreateBuffer()

	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.VertexAttribPointer(vertexPosition, 3, gl.FLOAT, false, 0, pc.Stride())
	gl.EnableVertexAttribArray(vertexPosition)

	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)

	return pc.Points - 1
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
