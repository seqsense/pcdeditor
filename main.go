package main

import (
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
	gl.Canvas.OnClick(func(e webgl.MouseEvent) {
		e.PreventDefault()
		chClick <- e
	})
	chMouseDown := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseDown(func(e webgl.MouseEvent) {
		e.PreventDefault()
		chMouseDown <- e
	})
	chMouseMove := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseMove(func(e webgl.MouseEvent) {
		e.PreventDefault()
		chMouseMove <- e
	})
	chMouseUp := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseUp(func(e webgl.MouseEvent) {
		e.PreventDefault()
		chMouseUp <- e
	})

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

	vi := &view{}

	for {
		newWidth := gl.Canvas.ClientWidth()
		newHeight := gl.Canvas.ClientHeight()
		if newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			updateProjectionMatrix(width, height)
		}

		modelViewMatrix :=
			modelViewMatrixBase.
				Mul(mat.Rotate(0, 0, 1, float32(vi.ang))).
				Mul(mat.Translate(float32(vi.x), float32(vi.y), 0))

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
				vi.mouseDragStart(&e)
				continue
			case e := <-chMouseUp:
				vi.mouseDragEnd(&e)
				continue
			case e := <-chMouseMove:
				vi.mouseDrag(&e)
				continue
			case e := <-chClick:
				if e.Button == 0 && pc != nil {
					selected, ok := selectPoint(
						pc, modelViewMatrix, projectionMatrix, fov, e.OffsetX, e.OffsetY, width, height,
					)
					if ok {
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
