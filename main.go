package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
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
	vsSel := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vsSel, vsSelectSource)
	gl.CompileShader(vsSel)
	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(fs, fsSource)
	gl.CompileShader(fs)

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	programSel := gl.CreateProgram()
	gl.AttachShader(programSel, vsSel)
	gl.AttachShader(programSel, fs)
	gl.LinkProgram(programSel)

	projectionMatrixLocation := gl.GetUniformLocation(program, "uProjectionMatrix")
	projectionMatrixLocationSel := gl.GetUniformLocation(programSel, "uProjectionMatrix")
	modelViewMatrixLocation := gl.GetUniformLocation(program, "uModelViewMatrix")
	modelViewMatrixLocationSel := gl.GetUniformLocation(programSel, "uModelViewMatrix")

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	posBuf := gl.CreateBuffer()

	const fov = 3.14 / 4
	var projectionMatrix mat.Mat4
	updateProjectionMatrix := func(width, height int) {
		gl.Canvas.SetWidth(width)
		gl.Canvas.SetHeight(height)
		projectionMatrix = mat.Perspective(
			fov,
			float32(width)/float32(height),
			1.0, 1000.0,
		)
		gl.UseProgram(program)
		gl.UniformMatrix4fv(projectionMatrixLocation, false, projectionMatrix)
		gl.UseProgram(programSel)
		gl.UniformMatrix4fv(projectionMatrixLocationSel, false, projectionMatrix)
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

	chUpdateView := make(chan float64)
	viewDistance := 100.0
	var modelViewMatrixBase mat.Mat4
	updateView := func(distance float64) {
		modelViewMatrixBase =
			mat.Translate(0, 0, -float32(distance)).
				Mul(mat.Rotate(1, 0, 0, 3.14/4))
	}
	updateView(viewDistance)

	gl.Canvas.OnWheel(func(e webgl.WheelEvent) {
		e.PreventDefault()
		viewDistance += e.DeltaY
		chUpdateView <- viewDistance
	})
	chClick := make(chan webgl.MouseEvent)
	gl.Canvas.OnClick(func(e webgl.MouseEvent) { chClick <- e })
	chMouseDown := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseDown(func(e webgl.MouseEvent) { chMouseDown <- e })
	chMouseMove := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseMove(func(e webgl.MouseEvent) { chMouseMove <- e })
	chMouseUp := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseUp(func(e webgl.MouseEvent) { chMouseUp <- e })

	toolBuf := gl.CreateBuffer()

	updateCursor := func(v0, v1 mat.Vec3) {
		gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
		gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer([]float32{
			v0[0], v0[1], v0[2],
			v1[0], v1[1], v1[2],
			0, 0, 0,
		}), gl.STATIC_DRAW)
	}
	updateCursor(mat.NewVec3(0, 0, 0), mat.NewVec3(0, 0, 20))

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	var nPoints int
	var pc *pcd.PointCloud

	gl.UseProgram(program)
	vertexPosition := gl.GetAttribLocation(program, "aVertexPosition")
	gl.EnableVertexAttribArray(vertexPosition)

	gl.UseProgram(programSel)
	vertexPositionSel := gl.GetAttribLocation(programSel, "aVertexPosition")
	gl.EnableVertexAttribArray(vertexPositionSel)

	var drugStart *webgl.MouseEvent
	angDragStart := ang
	for {
		newWidth := gl.Canvas.ClientWidth()
		newHeight := gl.Canvas.ClientHeight()
		if newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			updateProjectionMatrix(width, height)
		}

		modelViewMatrix := modelViewMatrixBase.Mul(mat.Rotate(0, 0, 1, ang))

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if nPoints > 0 {
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.VertexAttribPointer(vertexPosition, 3, gl.FLOAT, false, pc.Stride(), 0)

			gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)
			gl.DrawArrays(gl.POINTS, 0, nPoints)
		}

		gl.UseProgram(programSel)
		gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
		gl.VertexAttribPointer(vertexPositionSel, 3, gl.FLOAT, false, 0, 0)

		gl.UniformMatrix4fv(modelViewMatrixLocationSel, false, modelViewMatrix)
		gl.DrawArrays(gl.LINES, 0, 2)

		for {
			select {
			case path := <-chNewPath:
				logPrint("loading pcd file")
				p, n, err := loadPCD(gl, program, posBuf, path)
				if err != nil {
					logPrint(err)
					continue
				}
				logPrint("pcd file loaded")
				nPoints = n
				pc = p
				continue
			case d := <-chUpdateView:
				updateView(d)
				continue
			case e := <-chMouseDown:
				if e.Button == 0 {
					drugStart = &e
					angDragStart = ang
				}
				continue
			case e := <-chMouseUp:
				if drugStart != nil && e.Button == 0 {
					xDiff := e.ClientX - drugStart.ClientX
					ang = angDragStart - 0.02*float32(xDiff)
					drugStart = nil
				}
				continue
			case e := <-chMouseMove:
				if drugStart != nil {
					xDiff := e.ClientX - drugStart.ClientX
					ang = angDragStart - 0.02*float32(xDiff)
				}
				continue
			case e := <-chClick:
				if e.Button == 0 {
					pos := mat.NewVec3(
						float32(e.ClientX)*2/float32(width)-1,
						1-float32(e.ClientY)*2/float32(height), -1)

					a := projectionMatrix.Mul(modelViewMatrix).InvAffine()
					origin := a.Transform(mat.NewVec3(0, 0, -1-1.0/float32(math.Tan(fov))))
					target := a.Transform(pos)

					dir := target.Sub(origin).Normalized()

					if pc == nil {
						continue
					}
					it, err := pc.Float32Iterators("x", "y", "z")
					if err != nil {
						continue
					}
					xi, yi, zi := it[0], it[1], it[2]
					var selected *mat.Vec3
					dSqMin := float32(0.1 * 0.1)
					vMin := float32(1000 * 1000)
					for xi.IsValid() {
						p := mat.NewVec3(xi.Float32(), yi.Float32(), zi.Float32())
						pRel := origin.Sub(p)
						dot := pRel.Dot(dir)
						if dot < 0 {
							distSq := pRel.NormSq()
							dSq := distSq - dot*dot
							v := distSq/10000 + dSq
							if dSq < dSqMin && v < vMin {
								vMin = v
								selected = &p
							}
						}
						xi.Incr()
						yi.Incr()
						zi.Incr()
					}
					if selected != nil {
						updateCursor(*selected, mat.NewVec3(0, 0, 1).Add(*selected))
					}
				}
				continue
			case <-tick.C:
			}
			break
		}
	}
}

func loadPCD(gl *webgl.WebGL, program webgl.Program, buf webgl.Buffer, path string) (*pcd.PointCloud, int, error) {
	var b []byte
	chErr := make(chan error)
	js.Global().Call("fetch", path).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return args[0].Call("arrayBuffer")
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to fetch file")
			return nil
		}),
	).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			array := js.Global().Get("Uint8Array").New(args[0])
			n := array.Get("byteLength").Int()
			b = make([]byte, n)
			js.CopyBytesToGo(b, array)
			chErr <- nil
			return nil
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chErr <- errors.New("failed to handle received data")
			return nil
		}),
	)

	if err := <-chErr; err != nil {
		return nil, 0, err
	}

	pc, err := pcd.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)

	return pc, pc.Points, nil
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

		c = (aVertexPosition[2] - zMin) / zRange;
		vColor = vec4(c, 0.0, 1.0 - c, 1.0);
	}
`

const vsSelectSource = `
	attribute vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	varying lowp vec4 vColor;
	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 3.0;

		vColor = vec4(1.0, 1.0, 1.0, 0.8);
	}
`

const fsSource = `
	varying lowp vec4 vColor;
	void main(void) {
		gl_FragColor = vColor;
	}
`
