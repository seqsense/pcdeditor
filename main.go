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

	tick := time.NewTicker(time.Second / 5)
	defer tick.Stop()

	chNewPath := make(chan string)
	js.Global().Set("loadPCD",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chNewPath <- args[0].String()
			return nil
		}),
	)

	chWheel := make(chan webgl.WheelEvent)
	gl.Canvas.OnWheel(func(e webgl.WheelEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chWheel <- e
	})
	chClick := make(chan webgl.MouseEvent)
	gl.Canvas.OnClick(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chClick <- e
	})
	chMouseDown := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseDown(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chMouseDown <- e
	})
	chMouseMove := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseMove(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chMouseMove <- e
	})
	chMouseUp := make(chan webgl.MouseEvent)
	gl.Canvas.OnMouseUp(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chMouseUp <- e
	})
	gl.Canvas.OnContextMenu(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
	})
	chKey := make(chan webgl.KeyboardEvent)
	gl.Canvas.OnKeyDown(func(e webgl.KeyboardEvent) {
		e.PreventDefault()
		e.StopPropagation()
		chKey <- e
	})

	toolBuf := gl.CreateBuffer()

	var nCursorPoints int
	updateCursor := func(pp ...mat.Vec3) {
		nCursorPoints = len(pp)
		buf := make([]float32, 0, len(pp)*3)
		for _, p := range pp {
			buf = append(buf, p[0], p[1], p[2])
		}
		if nCursorPoints > 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
			gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer(buf), gl.STATIC_DRAW)
		}
	}

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	var nPoints int
	var pc *pcd.PointCloud
	var selected []mat.Vec3

	gl.UseProgram(program)
	vertexPosition := gl.GetAttribLocation(program, "aVertexPosition")
	gl.EnableVertexAttribArray(vertexPosition)

	gl.UseProgram(programSel)
	vertexPositionSel := gl.GetAttribLocation(programSel, "aVertexPosition")
	gl.EnableVertexAttribArray(vertexPositionSel)

	vi := newView()
	cg := &clickGuard{}

	for {
		newWidth := gl.Canvas.ClientWidth()
		newHeight := gl.Canvas.ClientHeight()
		if newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			updateProjectionMatrix(width, height)
		}

		modelViewMatrixBase :=
			mat.Translate(0, 0, -float32(vi.distance)).
				Mul(mat.Rotate(1, 0, 0, float32(vi.pitch)))
		modelViewMatrix :=
			modelViewMatrixBase.
				Mul(mat.Rotate(0, 0, 1, float32(vi.yaw))).
				Mul(mat.Translate(float32(vi.x), float32(vi.y), 0))

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if nPoints > 0 {
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.VertexAttribPointer(vertexPosition, 3, gl.FLOAT, false, pc.Stride(), 0)

			gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)
			gl.DrawArrays(gl.POINTS, 0, nPoints)
		}

		if nCursorPoints > 0 {
			gl.UseProgram(programSel)
			gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
			gl.VertexAttribPointer(vertexPositionSel, 3, gl.FLOAT, false, 0, 0)

			gl.UniformMatrix4fv(modelViewMatrixLocationSel, false, modelViewMatrix)
			gl.DrawArrays(gl.LINE_LOOP, 0, nCursorPoints)
			gl.DrawArrays(gl.POINTS, 0, nCursorPoints)
		}

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
		case e := <-chWheel:
			vi.wheel(&e)
		case e := <-chMouseDown:
			vi.mouseDragStart(&e)
			if e.Button == 0 {
				cg.DragStart()
			}
		case e := <-chMouseUp:
			vi.mouseDragEnd(&e)
			if e.Button == 0 {
				cg.DragEnd()
			}
		case e := <-chMouseMove:
			vi.mouseDrag(&e)
			cg.Move()
		case e := <-chClick:
			if e.Button == 0 && pc != nil && cg.Click() {
				p, ok := selectPoint(
					pc, modelViewMatrix, projectionMatrix, fov, e.OffsetX, e.OffsetY, width, height,
				)
				if ok {
					switch {
					case e.ShiftKey:
						if len(selected) > 0 {
							if len(selected) > 1 {
								selected[1] = *p
							} else {
								selected = append(selected, *p)
							}
						}
					default:
						if len(selected) < 2 {
							selected = []mat.Vec3{*p}
						} else {
							if len(selected) > 2 {
								selected[2] = *p
							} else {
								selected = append(selected, *p)
							}
						}
					}
					switch len(selected) {
					case 3:
						base := selected[1].Sub(selected[0])
						proj := selected[0].Add(
							base.Mul(base.Dot(selected[2].Sub(selected[0])) / base.NormSq()))
						perp := selected[2].Sub(proj)
						p2 := selected[1].Add(perp)
						p3 := selected[0].Add(perp)
						updateCursor(selected[0], selected[1], p2, p3)
					default:
						updateCursor(selected...)
					}
				}
			}
			gl.Canvas.Focus()
		case e := <-chKey:
			switch e.Code {
			case "Escape":
				selected = nil
				updateCursor()
			}
		case <-tick.C:
		}
	}
}
