package main

import (
	"fmt"
	"math"
	"syscall/js"
	"time"

	webgl "github.com/seqsense/pcdeditor/gl"
	"github.com/seqsense/pcdeditor/mat"
	"github.com/seqsense/pcdeditor/pcd"
)

const (
	vib3DXAmp = 0.002
)

func main() {
	cb := js.Global().Get("document").Get("onPCDEditorLoaded")
	if !cb.IsNull() {
		cb.Invoke(
			map[string]interface{}{
				"attach": js.FuncOf(newPCDEditor),
			},
		)
	}
	select {}
}

type promiseCommand struct {
	data     string
	resolved func(string)
	rejected func(error)
}

type pcdeditor struct {
	canvas      js.Value
	logPrint    func(msg interface{})
	chLoadPath  chan promiseCommand
	chSavePath  chan promiseCommand
	chExport    chan promiseCommand
	chCommand   chan promiseCommand
	chWheel     chan webgl.WheelEvent
	chClick     chan webgl.MouseEvent
	chMouseDown chan webgl.MouseEvent
	chMouseMove chan webgl.MouseEvent
	chMouseUp   chan webgl.MouseEvent
	chKey       chan webgl.KeyboardEvent
}

func newPCDEditor(this js.Value, args []js.Value) interface{} {
	canvas := args[0]
	pe := &pcdeditor{
		canvas: canvas,
		logPrint: func(msg interface{}) {
			fmt.Println(msg)
		},
		chLoadPath:  make(chan promiseCommand, 1),
		chSavePath:  make(chan promiseCommand, 1),
		chExport:    make(chan promiseCommand, 1),
		chCommand:   make(chan promiseCommand, 1),
		chWheel:     make(chan webgl.WheelEvent, 10),
		chClick:     make(chan webgl.MouseEvent, 10),
		chMouseDown: make(chan webgl.MouseEvent, 10),
		chMouseMove: make(chan webgl.MouseEvent, 10),
		chMouseUp:   make(chan webgl.MouseEvent, 10),
		chKey:       make(chan webgl.KeyboardEvent, 10),
	}
	if len(args) > 1 {
		init := args[1]
		if logger := init.Get("logger"); !logger.IsNull() {
			pe.logPrint = func(msg interface{}) {
				logger.Invoke(fmt.Sprintf("%v", msg))
			}
		}
	}
	go pe.Run()

	return js.ValueOf(map[string]interface{}{
		"loadPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chLoadPath, args[0].String())
		}),
		"savePCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chSavePath, args[0].String())
		}),
		"exportPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chExport, args[0].String())
		}),
		"command": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chCommand, args[0].String())
		}),
	})
}
func newCommandPromise(ch chan promiseCommand, data string) js.Value {
	promise := js.Global().Get("Promise")
	return promise.New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve, reject := args[0], args[1]
		cmd := promiseCommand{
			data:     data,
			resolved: func(msg string) { resolve.Invoke(msg) },
			rejected: func(err error) { reject.Invoke(errorToJS(err)) },
		}
		select {
		case ch <- cmd:
			return nil
		default:
			reject.Invoke()
			return nil
		}
	}))
}

func (pe *pcdeditor) Run() {
	gl, err := webgl.New(pe.canvas)
	if err != nil {
		pe.logPrint(err)
		return
	}

	vs := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vs, vsSource)
	gl.CompileShader(vs)
	if !gl.GetShaderParameter(vs, gl.COMPILE_STATUS).(bool) {
		pe.logPrint("Compile failed (VERTEX_SHADER)")
		return
	}
	vsSel := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vsSel, vsSelectSource)
	gl.CompileShader(vsSel)
	if !gl.GetShaderParameter(vsSel, gl.COMPILE_STATUS).(bool) {
		pe.logPrint("Compile failed (VERTEX_SHADER)")
		return
	}
	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(fs, fsSource)
	gl.CompileShader(fs)
	if !gl.GetShaderParameter(fs, gl.COMPILE_STATUS).(bool) {
		pe.logPrint("Compile failed (FRAGMENT_SHADER)")
		return
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)
	if !gl.GetProgramParameter(program, gl.LINK_STATUS).(bool) {
		pe.logPrint("Link failed: " + gl.GetProgramInfoLog(program))
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

	fov := math.Pi / 3
	var projectionMatrix mat.Mat4
	updateProjectionMatrix := func(width, height int) {
		gl.Canvas.SetWidth(width)
		gl.Canvas.SetHeight(height)
		projectionMatrix = mat.Perspective(
			float32(fov),
			float32(width)/float32(height),
			1.0, 1000.0,
		)
		gl.UseProgram(program)
		gl.UniformMatrix4fv(projectionMatrixLocation, false, projectionMatrix)
		gl.UseProgram(programSel)
		gl.UniformMatrix4fv(projectionMatrixLocationSel, false, projectionMatrix)
		gl.Viewport(0, 0, width, height)
	}
	var width, height int

	tick := time.NewTicker(time.Second / 8)
	defer tick.Stop()

	var vib3D bool
	var vib3DX float32

	gl.Canvas.OnClick(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chClick <- e:
		default:
		}
	})
	gl.Canvas.OnContextMenu(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
	})
	gl.Canvas.OnKeyDown(func(e webgl.KeyboardEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chKey <- e:
		default:
		}
	})

	wheelHandler := func(e webgl.WheelEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chWheel <- e:
		default:
		}
	}
	gl.Canvas.OnWheel(wheelHandler)
	gesture := &gesture{
		pointers: make(map[int]webgl.PointerEvent),
		onMouseDown: func(e webgl.MouseEvent) {
			select {
			case pe.chMouseDown <- e:
			default:
			}
		},
		onMouseMove: func(e webgl.MouseEvent) {
			select {
			case pe.chMouseMove <- e:
			default:
			}
		},
		onMouseUp: func(e webgl.MouseEvent) {
			select {
			case pe.chMouseUp <- e:
			default:
			}
		},
		onWheel: wheelHandler,
	}
	gl.Canvas.OnPointerDown(gesture.pointerDown)
	gl.Canvas.OnPointerMove(gesture.pointerMove)
	gl.Canvas.OnPointerUp(gesture.pointerUp)
	gl.Canvas.OnPointerOut(gesture.pointerUp)

	toolBuf := gl.CreateBuffer()

	var nRectPoints int
	cmd := newCommandContext(&pcdIOImpl{})
	cs := &console{cmd: cmd}

	loadPoints := func(gl *webgl.WebGL, buf webgl.Buffer, pc *pcd.PointCloud) {
		if pc.Points > 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, buf)
			gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)
		}
	}

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

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

	devicePixelRatioJS := js.Global().Get("window").Get("devicePixelRatio")

	// Allow export after crash
	defer func() {
		if r := recover(); r != nil {
			pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			pe.logPrint(r)
			if pc, ok := cmd.PointCloud(); ok {
				pe.logPrint("CRASHED (export command is available)")
				pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				for promise := range pe.chExport {
					err := cmd.pcdIO.exportPCD(promise.data, pc)
					if err != nil {
						promise.rejected(err)
						continue
					}
					promise.resolved("exported")
				}
			} else {
				pe.logPrint("CRASHED")
				pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			}
		}
	}()

	for {
		scale := devicePixelRatioJS.Int()
		newWidth := gl.Canvas.ClientWidth() * scale
		newHeight := gl.Canvas.ClientHeight() * scale
		if newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			updateProjectionMatrix(width, height)
		}

		modelViewMatrix :=
			mat.Translate(vib3DX, 0, -float32(vi.distance)).
				MulAffine(mat.Rotate(1, 0, 0, float32(vi.pitch))).
				MulAffine(mat.Rotate(0, 0, 1, float32(vi.yaw))).
				MulAffine(mat.Translate(float32(vi.x), float32(vi.y), -1.5))

		if rect, updated := cmd.Rect(); updated {
			buf := make([]float32, 0, len(rect)*3)
			for _, p := range rect {
				buf = append(buf, p[0], p[1], p[2])
			}
			nRectPoints = len(rect)
			if len(rect) > 0 {
				gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
				gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer(buf), gl.STATIC_DRAW)
			}
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if pc, ok := cmd.PointCloud(); ok && pc.Points > 0 {
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, pc.Stride(), 0)
			gl.VertexAttribIPointer(aVertexLabel, 1, gl.UNSIGNED_INT, pc.Stride(), 3*4)

			gl.UniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix)
			gl.DrawArrays(gl.POINTS, 0, pc.Points-1)
		}

		if nRectPoints > 0 {
			gl.UseProgram(programSel)
			for i := 0; i < nRectPoints; i += 4 {
				gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
				gl.VertexAttribPointer(aVertexPositionSel, 3, gl.FLOAT, false, 3*4, 3*4*i)
				n := 4
				if n > nRectPoints-i {
					n = nRectPoints - i
				}

				gl.UniformMatrix4fv(modelViewMatrixLocationSel, false, modelViewMatrix)
				gl.DrawArrays(gl.LINE_LOOP, 0, n)
				gl.DrawArrays(gl.POINTS, 0, n)
			}
		}

		select {
		case promise := <-pe.chLoadPath:
			pe.logPrint("loading pcd file")
			if err := cmd.LoadPCD(promise.data); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file loaded")
			pc, _ := cmd.PointCloud()
			loadPoints(gl, posBuf, pc)
			promise.resolved("loaded")
		case promise := <-pe.chSavePath:
			pe.logPrint("saving pcd file")
			if err := cmd.SavePCD(promise.data); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file saved")
			promise.resolved("saved")
		case promise := <-pe.chExport:
			pe.logPrint("exporting pcd file")
			if err := cmd.ExportPCD(promise.data); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file exported")
			promise.resolved("exported")
		case promise := <-pe.chCommand:
			res, err := cs.Run(promise.data)
			if err != nil {
				promise.rejected(err)
				break
			}
			promise.resolved(res)
		case e := <-pe.chWheel:
			switch {
			case e.CtrlKey:
				cmd.SetSelectRange(cmd.SelectRange() + float32(e.DeltaY)*0.01)
			case e.ShiftKey:
				rect := cmd.RectCenter()
				if len(rect) == 4 {
					c := rect[0].Add(rect[1]).Add(rect[2]).Add(rect[3]).Mul(1.0 / 4.0)
					r := 1.0 + float32(e.DeltaY)*0.01
					cmd.TransformCursors(
						mat.Translate(c[0], c[1], c[2]).
							MulAffine(mat.Scale(r, r, r)).
							MulAffine(mat.Translate(-c[0], -c[1], -c[2])),
					)
				}
			default:
				vi.wheel(&e)
			}
		case e := <-pe.chMouseDown:
			vi.mouseDragStart(&e)
			if e.Button == 0 {
				cg.DragStart()
			}
		case e := <-pe.chMouseUp:
			vi.mouseDragEnd(&e)
			if e.Button == 0 {
				cg.DragEnd()
			}
		case e := <-pe.chMouseMove:
			vi.mouseDrag(&e)
			cg.Move()
		case e := <-pe.chClick:
			if pc, ok := cmd.PointCloud(); ok && e.Button == 0 && cg.Click() {
				p, ok := selectPoint(
					pc, modelViewMatrix, projectionMatrix, e.OffsetX*scale, e.OffsetY*scale, width, height,
				)
				if ok {
					switch {
					case e.ShiftKey:
						cmd.SetCursor(1, *p)
					default:
						if len(cmd.Cursors()) < 2 {
							cmd.SetCursor(0, *p)
						} else {
							cmd.SetCursor(2, *p)
						}
					}
				}
			}
			gl.Canvas.Focus()
		case e := <-pe.chKey:
			switch e.Code {
			case "Escape":
				cmd.UnsetCursors()
			case "Delete", "Digit0", "Digit1":
				switch e.Code {
				case "Delete":
					if cmd.Delete() {
						pc, _ := cmd.PointCloud()
						loadPoints(gl, posBuf, pc)
					}
					if !e.ShiftKey && !e.CtrlKey {
						cmd.UnsetCursors()
					}
				case "Digit0", "Digit1":
					var l uint32
					if e.Code == "Digit1" {
						l = 1
					}
					if cmd.Label(l) {
						pc, _ := cmd.PointCloud()
						loadPoints(gl, posBuf, pc)
					}
				}
			case "KeyU":
				if cmd.Undo() {
					pc, _ := cmd.PointCloud()
					loadPoints(gl, posBuf, pc)
				}
			case "KeyF":
				if cmd.AddSurface(defaultResolution) {
					pc, _ := cmd.PointCloud()
					loadPoints(gl, posBuf, pc)
				}
			case "KeyV", "KeyH":
				switch e.Code {
				case "KeyV":
					cmd.SnapVertical()
				case "KeyH":
					cmd.SnapHorizontal()
				}
			case "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "PageUp", "PageDown":
				var dx, dy, dz float32
				switch e.Code {
				case "ArrowUp":
					dy = 0.05
				case "ArrowDown":
					dy = -0.05
				case "ArrowLeft":
					dx = -0.05
				case "ArrowRight":
					dx = 0.05
				case "PageUp":
					dz = 0.05
				case "PageDown":
					dz = -0.05
				}
				s, c := math.Sincos(vi.yaw)
				cmd.TransformCursors(mat.Translate(
					float32(c)*dx-float32(s)*dy,
					float32(s)*dx+float32(c)*dy,
					dz,
				))
			case "KeyW", "KeyA", "KeyS", "KeyD", "KeyQ", "KeyE":
				switch e.Code {
				case "KeyW":
					vi.move(0.05, 0, 0)
				case "KeyA":
					vi.move(0, 0.05, 0)
				case "KeyS":
					vi.move(-0.05, 0, 0)
				case "KeyD":
					vi.move(0, -0.05, 0)
				case "KeyQ":
					vi.move(0, 0, 0.02)
				case "KeyE":
					vi.move(0, 0, -0.02)
				}
			case "BracketRight", "Backslash":
				switch e.Code {
				case "BracketRight":
					fov += math.Pi / 16
					if fov > math.Pi*2/3 {
						fov = math.Pi * 2 / 3
					}
				case "Backslash":
					fov -= math.Pi / 16
					if fov < math.Pi/8 {
						fov = math.Pi / 8
					}
				}
				updateProjectionMatrix(width, height)
			case "F1":
				vi.reset()
			case "F2":
				vi.fps()
			case "F11":
				vi.snapYaw()
			case "F12":
				vi.snapPitch()
			case "KeyP":
				vib3D = !vib3D
			}
		case <-tick.C:
			if vib3D {
				if vib3DX < 0 {
					vib3DX = vib3DXAmp
				} else {
					vib3DX = -vib3DXAmp
				}
			} else {
				vib3DX = 0
			}
		}
	}
}
