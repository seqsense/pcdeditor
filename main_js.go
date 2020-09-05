package main

import (
	"fmt"
	"syscall/js"
	"time"

	webgl "github.com/seqsense/pcdeditor/gl"
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

const (
	resolution = 0.05
	delRange   = 0.1
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
	if !gl.GetShaderParameter(vs, gl.COMPILE_STATUS).(bool) {
		logPrint("Compile failed (VERTEX_SHADER)")
		return
	}
	vsSel := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vsSel, vsSelectSource)
	gl.CompileShader(vsSel)
	if !gl.GetShaderParameter(vsSel, gl.COMPILE_STATUS).(bool) {
		logPrint("Compile failed (VERTEX_SHADER)")
		return
	}
	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(fs, fsSource)
	gl.CompileShader(fs)
	if !gl.GetShaderParameter(fs, gl.COMPILE_STATUS).(bool) {
		logPrint("Compile failed (FRAGMENT_SHADER)")
		return
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)
	if !gl.GetProgramParameter(program, gl.LINK_STATUS).(bool) {
		logPrint("Link failed: " + gl.GetProgramInfoLog(program))
		return
	}

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
	chSavePath := make(chan string)
	js.Global().Set("savePCD",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chSavePath <- args[0].String()
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

	loadPoints := func(gl *webgl.WebGL, buf webgl.Buffer, pc *pcd.PointCloud) {
		if pc.Points > 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, buf)
			gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)
		}
	}

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	edit := &editor{}
	var selected []mat.Vec3

	gl.UseProgram(program)
	aVertexPosition := 0
	aVertexLabel := 1
	gl.EnableVertexAttribArray(aVertexPosition)
	gl.EnableVertexAttribArray(aVertexLabel)

	gl.UseProgram(programSel)
	aVertexPositionSel := 0
	gl.EnableVertexAttribArray(aVertexPositionSel)

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
				MulAffine(mat.Rotate(1, 0, 0, float32(vi.pitch)))
		modelViewMatrix :=
			modelViewMatrixBase.
				Mul(mat.Rotate(0, 0, 1, float32(vi.yaw))).
				Mul(mat.Translate(float32(vi.x), float32(vi.y), 0))

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if edit.pc != nil && edit.pc.Points > 0 {
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, edit.pc.Stride(), 0)
			gl.VertexAttribIPointer(aVertexLabel, 1, gl.UNSIGNED_INT, edit.pc.Stride(), 3*4)

			gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)
			gl.DrawArrays(gl.POINTS, 0, edit.pc.Points-1)
		}

		if nCursorPoints > 0 {
			gl.UseProgram(programSel)
			gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
			gl.VertexAttribPointer(aVertexPositionSel, 3, gl.FLOAT, false, 0, 0)

			gl.UniformMatrix4fv(modelViewMatrixLocationSel, false, modelViewMatrix)
			gl.DrawArrays(gl.LINE_LOOP, 0, nCursorPoints)
			gl.DrawArrays(gl.POINTS, 0, nCursorPoints)
		}

		select {
		case path := <-chNewPath:
			logPrint("loading pcd file")
			p, err := readPCD(path)
			if err != nil {
				logPrint(err)
				continue
			}
			if err := edit.Set(p); err != nil {
				logPrint(err)
				continue
			}
			loadPoints(gl, posBuf, edit.pc)
			logPrint("pcd file loaded")
		case path := <-chSavePath:
			if edit.pc != nil {
				logPrint("saving pcd file")
				err := writePCD(path, edit.pc)
				if err != nil {
					logPrint(err)
					continue
				}
				logPrint("pcd file saved")
			}
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
			if e.Button == 0 && edit.pc != nil && cg.Click() {
				p, ok := selectPoint(
					edit.pc, modelViewMatrix, projectionMatrix, fov, e.OffsetX, e.OffsetY, width, height,
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
						p2, p3 := rectFrom3(selected[0], selected[1], selected[2])
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
			case "Delete", "Digit0", "Digit1":
				if len(selected) != 3 {
					break
				}
				p0, p1 := selected[0], selected[1]
				_, p2 := rectFrom3(p0, p1, selected[2])
				v0, v1 := p1.Sub(p0), p2.Sub(p0)
				v0n, v1n := v0.Normalized(), v1.Normalized()
				v2n := v0n.Cross(v1n)
				m := (mat.Mat4{
					v0n[0], v0n[1], v0n[2], 0,
					v1n[0], v1n[1], v1n[2], 0,
					v2n[0], v2n[1], v2n[2], 0,
					0, 0, 0, 1,
				}).InvAffine().MulAffine(mat.Translate(-p0[0], -p0[1], -p0[2]))
				l0 := v0.Norm()
				l1 := v1.Norm()

				filter := func(p mat.Vec3) bool {
					if z := m.TransformZ(p); z < -delRange || delRange < z {
						return true
					}
					if x := m.TransformX(p); x < 0 || l0 < x {
						return true
					}
					if y := m.TransformY(p); y < 0 || l1 < y {
						return true
					}
					return false
				}
				switch e.Code {
				case "Delete":
					edit.Filter(filter)
					selected = nil
					updateCursor()
				case "Digit0", "Digit1":
					var l uint32
					if e.Code == "Digit1" {
						l = 1
					}
					edit.Label(func(p mat.Vec3) (uint32, bool) {
						if filter(p) {
							return 0, false
						}
						return l, true
					})
				}
				loadPoints(gl, posBuf, edit.pc)
			case "KeyF":
				if len(selected) != 3 {
					break
				}
				p0, p1 := selected[0], selected[1]
				_, p2 := rectFrom3(p0, p1, selected[2])
				v0, v1 := p1.Sub(p0), p2.Sub(p0)
				v0n, v1n := v0.Normalized(), v1.Normalized()
				v2n := v0n.Cross(v1n)
				m := mat.Translate(p0[0], p0[1], p0[2]).MulAffine(mat.Mat4{
					v0n[0], v0n[1], v0n[2], 0,
					v1n[0], v1n[1], v1n[2], 0,
					v2n[0], v2n[1], v2n[2], 0,
					0, 0, 0, 1,
				})
				l0 := v0.Norm()
				l1 := v1.Norm()

				w := int(l0 / resolution)
				h := int(l1 / resolution)
				pcNew := &pcd.PointCloud{
					PointCloudHeader: edit.pc.PointCloudHeader.Clone(),
					Points:           w * h,
					Data:             make([]byte, w*h*edit.pc.Stride()),
				}
				it, err := pcNew.Vec3Iterator()
				if err != nil {
					break
				}
				for x := 0; x < w; x++ {
					for y := 0; y < h; y++ {
						it.SetVec3(
							m.Transform(mat.Vec3{float32(x) * resolution, float32(y) * resolution, 0}),
						)
						it.Incr()
					}
				}
				edit.Merge(pcNew)
				loadPoints(gl, posBuf, edit.pc)
			case "KeyV", "KeyH":
				if len(selected) != 3 {
					break
				}
				switch e.Code {
				case "KeyV":
					selected[2][0] = selected[0][0]
					selected[2][1] = selected[0][1]
				case "KeyH":
					selected[1][2] = selected[0][2]
					selected[2][2] = selected[0][2]
				}
				p2, p3 := rectFrom3(selected[0], selected[1], selected[2])
				updateCursor(selected[0], selected[1], p2, p3)
			case "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "PageUp", "PageDown":
				var dx, dy, dz float32
				switch e.Code {
				case "ArrowUp":
					dx = 0.05
				case "ArrowDown":
					dx = -0.05
				case "ArrowLeft":
					dy = 0.05
				case "ArrowRight":
					dy = -0.05
				case "PageUp":
					dz = 0.05
				case "PageDown":
					dz = -0.05
				}
				for i := range selected {
					selected[i][0] += dx
					selected[i][1] += dy
					selected[i][2] += dz
				}
				if len(selected) < 3 {
					updateCursor(selected...)
				} else {
					p2, p3 := rectFrom3(selected[0], selected[1], selected[2])
					updateCursor(selected[0], selected[1], p2, p3)
				}
			}
		case <-tick.C:
		}
	}
}
