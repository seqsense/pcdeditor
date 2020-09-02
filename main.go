package main

import (
	"math"
	"syscall/js"
	"time"

	webgl "github.com/at-wat/pcdviewer/gl"
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

	vertexPosition := gl.GetAttribLocation(program, "aVertexPosition")
	vertexColor := gl.GetAttribLocation(program, "aVertexColor")
	projectionMatrixLocation := gl.GetUniformLocation(program, "uProjectionMatrix")
	modelViewMatrixLocation := gl.GetUniformLocation(program, "uModelViewMatrix")

	pos := make([]float32, 0, 1024*1024*4)
	col := make([]float32, 0, 1024*1024*4)
	var nPoint int
	for x := float32(-1.0); x < 1.0; x += 0.001 {
		for y := float32(-1.0); y < 1.0; y += 0.001 {
			pos = append(pos, x)
			pos = append(pos, y)
			pos = append(pos, float32(math.Sqrt(float64(x*x+y*y))))

			col = append(col, (x+1)/2)
			col = append(col, (x+1)/2)
			col = append(col, 1)
			col = append(col, 1)

			nPoint++
		}
	}

	posBuf := gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer(pos), gl.STATIC_DRAW)

	colBuf := gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, colBuf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer(col), gl.STATIC_DRAW)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	gl.VertexAttribPointer(vertexPosition, 3, gl.FLOAT, false, 0, 0)
	gl.EnableVertexAttribArray(vertexPosition)

	gl.BindBuffer(gl.ARRAY_BUFFER, colBuf)
	gl.VertexAttribPointer(vertexColor, 4, gl.FLOAT, false, 0, 0)
	gl.EnableVertexAttribArray(vertexColor)

	gl.UseProgram(program)

	projectionMatrix := [16]float32{
		1.81, 0, 0, 0,
		0, 2.41, 0, 0,
		0, 0, -1, -1,
		0, 0, -0.20, 0,
	}
	gl.UniformMatrix4fv(projectionMatrixLocation, false, projectionMatrix)

	ang := 0.0
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()
	for {
		modelViewMatrix := [16]float32{
			float32(math.Sin(ang)), 0, -float32(math.Cos(ang)), 0,
			0, 1, 0, 0,
			float32(math.Cos(ang)), 0, float32(math.Sin(ang)), 0,
			0, 0, -6, 1,
		}
		gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.DrawArrays(gl.POINTS, 0, nPoint)

		<-tick.C
		ang += 0.01
	}
}

const vsSource = `
	attribute vec4 aVertexPosition;
	attribute vec4 aVertexColor;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	varying lowp vec4 vColor;
	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 2.0;
		vColor = aVertexColor;
	}
`
const fsSource = `
	varying lowp vec4 vColor;
	void main(void) {
		gl_FragColor = vColor;
	}
`
