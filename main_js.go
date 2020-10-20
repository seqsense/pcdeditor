package main

import (
	"fmt"
	"math"
	"runtime"
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
	data     interface{}
	resolved func(interface{})
	rejected func(error)
}

type pcdeditor struct {
	canvas      js.Value
	logPrint    func(msg interface{})
	chLoadPCD   chan promiseCommand
	chLoad2D    chan promiseCommand
	chSavePCD   chan promiseCommand
	chExportPCD chan promiseCommand
	chCommand   chan promiseCommand
	chWheel     chan webgl.WheelEvent
	chClick     chan webgl.MouseEvent
	chMouseDown chan webgl.MouseEvent
	chMouseMove chan webgl.MouseEvent
	chMouseUp   chan webgl.MouseEvent
	chKey       chan webgl.KeyboardEvent
	ch2D        chan promiseCommand
}

func newPCDEditor(this js.Value, args []js.Value) interface{} {
	canvas := args[0]
	pe := &pcdeditor{
		canvas: canvas,
		logPrint: func(msg interface{}) {
			fmt.Println(msg)
		},
		chLoadPCD:   make(chan promiseCommand, 1),
		chLoad2D:    make(chan promiseCommand, 1),
		chSavePCD:   make(chan promiseCommand, 1),
		chExportPCD: make(chan promiseCommand, 1),
		chCommand:   make(chan promiseCommand, 1),
		chWheel:     make(chan webgl.WheelEvent, 10),
		chClick:     make(chan webgl.MouseEvent, 10),
		chMouseDown: make(chan webgl.MouseEvent, 10),
		chMouseMove: make(chan webgl.MouseEvent, 10),
		chMouseUp:   make(chan webgl.MouseEvent, 10),
		chKey:       make(chan webgl.KeyboardEvent, 10),
		ch2D:        make(chan promiseCommand, 1),
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
			return newCommandPromise(pe.chLoadPCD, args[0].String())
		}),
		"load2D": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chLoad2D, [2]string{args[0].String(), args[1].String()})
		}),
		"savePCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chSavePCD, args[0].String())
		}),
		"exportPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chExportPCD, nil)
		}),
		"command": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chCommand, args[0].String())
		}),
		"show2D": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.ch2D, args[0].Bool())
		}),
	})
}
func newCommandPromise(ch chan promiseCommand, data interface{}) js.Value {
	promise := js.Global().Get("Promise")
	return promise.New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve, reject := args[0], args[1]
		cmd := promiseCommand{
			data:     data,
			resolved: func(res interface{}) { resolve.Invoke(res) },
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
	vsMap := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(vsMap, vsMapSource)
	gl.CompileShader(vsMap)
	if !gl.GetShaderParameter(vsMap, gl.COMPILE_STATUS).(bool) {
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
	fsMap := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(fsMap, fsMapSource)
	gl.CompileShader(fsMap)
	if !gl.GetShaderParameter(fsMap, gl.COMPILE_STATUS).(bool) {
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
	if !gl.GetProgramParameter(programSel, gl.LINK_STATUS).(bool) {
		pe.logPrint("Link failed: " + gl.GetProgramInfoLog(programSel))
		return
	}

	programMap := gl.CreateProgram()
	gl.AttachShader(programMap, vsMap)
	gl.AttachShader(programMap, fsMap)
	gl.LinkProgram(programMap)
	if !gl.GetProgramParameter(programMap, gl.LINK_STATUS).(bool) {
		pe.logPrint("Link failed: " + gl.GetProgramInfoLog(programMap))
		return
	}

	uProjectionMatrixLocation := gl.GetUniformLocation(program, "uProjectionMatrix")
	uModelViewMatrixLocation := gl.GetUniformLocation(program, "uModelViewMatrix")
	uSelectMatrixLocation := gl.GetUniformLocation(program, "uSelectMatrix")
	uZMinLocation := gl.GetUniformLocation(program, "uZMin")
	uZRangeLocation := gl.GetUniformLocation(program, "uZRange")
	uPointSizeBase := gl.GetUniformLocation(program, "uPointSizeBase")

	uProjectionMatrixLocationSel := gl.GetUniformLocation(programSel, "uProjectionMatrix")
	uModelViewMatrixLocationSel := gl.GetUniformLocation(programSel, "uModelViewMatrix")

	uProjectionMatrixLocationMap := gl.GetUniformLocation(programMap, "uProjectionMatrix")
	uModelViewMatrixLocationMap := gl.GetUniformLocation(programMap, "uModelViewMatrix")
	uSamplerLocationMap := gl.GetUniformLocation(programMap, "uSampler")
	uAlphaLocationMap := gl.GetUniformLocation(programMap, "uAlpha")

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	posBuf := gl.CreateBuffer()
	mapBuf := gl.CreateBuffer()

	tick := time.NewTicker(time.Second / 8)
	defer tick.Stop()

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

	fov := math.Pi / 3
	var prevFov float64
	var projectionMatrix mat.Mat4
	var width, height int
	var distance float64
	var projectionType ProjectionType

	var vib3D bool
	var vib3DX float32

	var nRectPoints int
	cmd := newCommandContext(&pcdIOImpl{}, &mapIOImpl{})
	cs := &console{cmd: cmd}

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	const (
		aVertexPosition    = 0
		aVertexLabel       = 1
		aVertexPositionSel = 0
		aVertexPositionMap = 0
		aTextureCoordMap   = 1
	)
	gl.UseProgram(program)
	gl.EnableVertexAttribArray(aVertexPosition)
	gl.EnableVertexAttribArray(aVertexLabel)

	gl.UseProgram(programSel)
	gl.EnableVertexAttribArray(aVertexPositionSel)

	gl.UseProgram(programMap)
	gl.EnableVertexAttribArray(aVertexPositionMap)
	gl.EnableVertexAttribArray(aTextureCoordMap)

	vi := newView()
	cg := &clickGuard{}

	devicePixelRatioJS := js.Global().Get("window").Get("devicePixelRatio")
	wheelNormalizer := &wheelNormalizer{}

	texture := gl.CreateTexture()
	mapRect := &pcd.PointCloud{
		PointCloudHeader: pcd.PointCloudHeader{
			Fields: []string{"x", "y", "z", "u", "v"},
			Size:   []int{4, 4, 4, 4, 4},
			Count:  []int{1, 1, 1, 1, 1},
		},
		Points: 5,
		Data:   make([]byte, 5*4*5),
	}
	var show2D bool = true
	var has2D bool

	// Allow export after crash
	defer func() {
		if r := recover(); r != nil {
			pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			for d := 0; ; d++ {
				ptr, file, line, ok := runtime.Caller(d)
				if !ok {
					break
				}
				f := runtime.FuncForPC(ptr)
				fmt.Printf("%s:%d: %s\n", file, line, f.Name())
			}
			pe.logPrint(r)
			if pc, _, ok := cmd.PointCloud(); ok {
				pe.logPrint("CRASHED (export command is available)")
				pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				for promise := range pe.chExportPCD {
					blob, err := cmd.pcdIO.exportPCD(pc)
					if err != nil {
						promise.rejected(err)
						continue
					}
					promise.resolved(blob)
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
		newProjectionType := cmd.ProjectionType()
		newDistance := vi.distance
		if newWidth != width || newHeight != height || fov != prevFov || projectionType != newProjectionType || (newProjectionType == ProjectionOrthographic && newDistance != distance) {
			width, height = newWidth, newHeight
			projectionType = newProjectionType
			distance = newDistance

			gl.Canvas.SetWidth(width)
			gl.Canvas.SetHeight(height)
			switch projectionType {
			case ProjectionPerspective:
				projectionMatrix = mat.Perspective(
					float32(fov),
					float32(width)/float32(height),
					1.0, 1000.0,
				)
				gl.Uniform1f(uPointSizeBase, 20.0)
			case ProjectionOrthographic:
				projectionMatrix = mat.Orthographic(
					-float32(width/2)*float32(distance)/1000,
					float32(width/2)*float32(distance)/1000,
					float32(height/2)*float32(distance)/1000,
					-float32(height/2)*float32(distance)/1000,
					-1000, 1000.0,
				)
				gl.Uniform1f(uPointSizeBase, 1.0)
			}
			gl.UseProgram(program)
			gl.UniformMatrix4fv(uProjectionMatrixLocation, false, projectionMatrix)
			gl.UseProgram(programSel)
			gl.UniformMatrix4fv(uProjectionMatrixLocationSel, false, projectionMatrix)
			gl.UseProgram(programMap)
			gl.UniformMatrix4fv(uProjectionMatrixLocationMap, false, projectionMatrix)
			gl.Viewport(0, 0, width, height)
		}
		prevFov = fov

		modelViewMatrix := mat.Rotate(1, 0, 0, float32(vi.pitch)).
			MulAffine(mat.Rotate(0, 0, 1, float32(vi.yaw))).
			MulAffine(mat.Translate(float32(vi.x), float32(vi.y), -1.5))
		if projectionType == ProjectionPerspective {
			modelViewMatrix =
				mat.Translate(vib3DX, 0, -float32(vi.distance)).MulAffine(modelViewMatrix)
		}

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

		if pc, updated, ok := cmd.PointCloudCropped(); ok && updated {
			if pc.Points > 0 {
				gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
				gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pc.Data), gl.STATIC_DRAW)

				mi, img, ok := cmd.Map()
				has2D = ok
				if ok {
					gl.BindTexture(gl.TEXTURE_2D, texture)
					gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img.Interface().(js.Value))
					gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
					gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
					gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
					gl.BindTexture(gl.TEXTURE_2D, webgl.Texture(nil))

					gl.UseProgram(programMap)
					gl.ActiveTexture(gl.TEXTURE0)
					gl.BindTexture(gl.TEXTURE_2D, texture)
					gl.Uniform1i(uSamplerLocationMap, 0)

					w, h := img.Width(), img.Height()
					xi, _ := mapRect.Float32Iterator("x")
					yi, _ := mapRect.Float32Iterator("y")
					ui, _ := mapRect.Float32Iterator("u")
					vi, _ := mapRect.Float32Iterator("v")
					push := func(x, y, u, v float32) {
						xi.SetFloat32(x)
						yi.SetFloat32(y)
						ui.SetFloat32(u)
						vi.SetFloat32(v)
						xi.Incr()
						yi.Incr()
						ui.Incr()
						vi.Incr()
					}
					push(mi.Origin[0], mi.Origin[1], 0, 1)
					push(mi.Origin[0]+float32(w)*mi.Resolution, mi.Origin[1], 1, 1)
					push(mi.Origin[0]+float32(w)*mi.Resolution, mi.Origin[1]+float32(h)*mi.Resolution, 1, 0)
					push(mi.Origin[0], mi.Origin[1]+float32(h)*mi.Resolution, 0, 0)
					push(mi.Origin[0], mi.Origin[1], 0, 1)

					gl.BindBuffer(gl.ARRAY_BUFFER, mapBuf)
					gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(mapRect.Data), gl.STATIC_DRAW)
				}
			}
		}

		gl.UseProgram(program)
		if m, ok := cmd.SelectMatrix(); ok {
			gl.UniformMatrix4fv(uSelectMatrixLocation, false, m)
		} else {
			gl.UniformMatrix4fv(uSelectMatrixLocation, false, mat.Mat4{})
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		var hasPointCloud bool
		if pc, _, ok := cmd.PointCloudCropped(); ok && pc.Points > 0 {
			hasPointCloud = true
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, pc.Stride(), 0)
			gl.VertexAttribIPointer(aVertexLabel, 1, gl.UNSIGNED_INT, pc.Stride(), 3*4)
			gl.UniformMatrix4fv(uModelViewMatrixLocation, false, modelViewMatrix)
			zMin, zMax := cmd.ZRange()
			gl.Uniform1f(uZMinLocation, zMin)
			gl.Uniform1f(uZRangeLocation, zMax-zMin)
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

				gl.UniformMatrix4fv(uModelViewMatrixLocationSel, false, modelViewMatrix)
				gl.DrawArrays(gl.LINE_LOOP, 0, n)
				gl.DrawArrays(gl.POINTS, 0, n)
			}
		}

		if hasPointCloud && show2D && has2D {
			gl.Enable(gl.BLEND)
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
			gl.UseProgram(programMap)
			gl.BindBuffer(gl.ARRAY_BUFFER, mapBuf)
			gl.VertexAttribPointer(aVertexPositionMap, 3, gl.FLOAT, false, mapRect.Stride(), 0)
			gl.VertexAttribPointer(aTextureCoordMap, 2, gl.FLOAT, false, mapRect.Stride(), 4*3)
			gl.UniformMatrix4fv(uModelViewMatrixLocationMap, false, modelViewMatrix)
			gl.Uniform1f(uAlphaLocationMap, cmd.MapAlpha())
			gl.DrawArrays(gl.TRIANGLE_FAN, 0, 5)
			gl.Disable(gl.BLEND)
		}

		select {
		case promise := <-pe.chLoadPCD:
			pe.logPrint("loading pcd file")
			if err := cmd.LoadPCD(promise.data.(string)); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file loaded")
			promise.resolved("loaded")
		case promise := <-pe.chLoad2D:
			pe.logPrint("loading 2D map file")
			paths := promise.data.([2]string)
			if err := cmd.Load2D(paths[0], paths[1]); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("2D map file loaded")
			promise.resolved("loaded")
		case promise := <-pe.chSavePCD:
			pe.logPrint("saving pcd file")
			if err := cmd.SavePCD(promise.data.(string)); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file saved")
			promise.resolved("saved")
		case promise := <-pe.chExportPCD:
			pe.logPrint("exporting pcd file")
			blob, err := cmd.ExportPCD()
			if err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd file exported")
			promise.resolved(blob)
		case promise := <-pe.chCommand:
			res, err := cs.Run(promise.data.(string))
			if err != nil {
				promise.rejected(err)
				break
			}
			promise.resolved(res)
		case promise := <-pe.ch2D:
			show2D = promise.data.(bool)
			promise.resolved("changed")
		case e := <-pe.chWheel:
			var ok bool
			e.DeltaY, ok = wheelNormalizer.Normalize(e.DeltaY)
			if !ok {
				break
			}
			switch {
			case e.CtrlKey:
				rate := 0.01
				if e.ShiftKey {
					rate = 0.1
				}
				if len(cmd.Cursors()) < 4 {
					cmd.SetSelectRange(cmd.SelectRange() + float32(e.DeltaY*rate))
					break
				}
				r := 1.0 + float32(e.DeltaY*rate)
				m, _ := cmd.SelectMatrix()
				cmd.TransformCursors(
					m.InvAffine().
						MulAffine(mat.Translate(0, 0, 0.5)).
						MulAffine(mat.Scale(1, 1, r)).
						MulAffine(mat.Translate(0, 0, -0.5)).
						MulAffine(m),
				)
			case e.ShiftKey:
				rect, _ := cmd.Rect()
				if len(rect) > 0 {
					var c mat.Vec3
					for _, p := range rect {
						c = c.Add(p)
					}
					c = c.Mul(1.0 / float32(len(rect)))
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
			if pc, _, ok := cmd.PointCloudCropped(); ok && e.Button == 0 && cg.Click() {
				var p *mat.Vec3
				switch projectionType {
				case ProjectionPerspective:
					p, ok = selectPoint(
						pc, projectionType, modelViewMatrix, projectionMatrix, e.OffsetX*scale, e.OffsetY*scale, width, height,
					)
				case ProjectionOrthographic:
					p = selectPointOrtho(
						modelViewMatrix, projectionMatrix, e.OffsetX*scale, e.OffsetY*scale, width, height,
					)
				default:
					ok = false
				}
				if ok {
					switch {
					case e.ShiftKey:
						if len(cmd.Cursors()) < 3 {
							cmd.SetCursor(1, *p)
						} else {
							cmd.SetCursor(3, *p)
						}
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
			case "Delete", "Backspace", "Digit0", "Digit1":
				switch e.Code {
				case "Delete", "Backspace":
					cmd.Delete()
					if !e.ShiftKey && !e.CtrlKey {
						cmd.UnsetCursors()
					}
				case "Digit0", "Digit1":
					var l uint32
					if e.Code == "Digit1" {
						l = 1
					}
					cmd.Label(l)
				}
			case "KeyU":
				cmd.Undo()
			case "KeyF":
				cmd.AddSurface(defaultResolution)
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
			case "F1":
				vi.reset()
			case "F2":
				vi.fps()
			case "F3":
				cmd.SetProjectionType(ProjectionPerspective)
			case "F4":
				cmd.SetProjectionType(ProjectionOrthographic)
			case "F10":
				cmd.Crop()
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
