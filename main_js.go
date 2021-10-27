package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"syscall/js"
	"time"

	"github.com/seqsense/pcgol/mat"
	"github.com/seqsense/pcgol/pc"
	webgl "github.com/seqsense/webgl-go"
)

const (
	vib3DXAmp = 0.002
)

var (
	Version   = "unknown"
	BuildDate = "unknown"
)

func main() {
	println("pcdeditor", Version, BuildDate)

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
	canvas              js.Value
	logPrint            func(msg interface{})
	chImportPCD         chan promiseCommand
	chImportSubPCD      chan promiseCommand
	chImport2D          chan promiseCommand
	chExportPCD         chan promiseCommand
	chExportSelectedPCD chan promiseCommand
	chReset             chan promiseCommand
	chCommand           chan promiseCommand
	chWheel             chan webgl.WheelEvent
	chClick             chan webgl.MouseEvent
	chMouseDown         chan webgl.MouseEvent
	chMouseDrag         chan webgl.MouseEvent
	chMouseMove         chan webgl.MouseEvent
	chMouseUp           chan webgl.MouseEvent
	chKey               chan webgl.KeyboardEvent
	ch2D                chan promiseCommand
	chContextLost       chan webgl.WebGLContextEvent
	chContextRestored   chan webgl.WebGLContextEvent

	vi  *viewImpl
	cg  *clickGuard
	cmd *commandContext
	cs  *console

	onKeyDownHook func(webgl.KeyboardEvent)
}

func newPCDEditor(this js.Value, args []js.Value) interface{} {
	canvas := args[0]
	pe := &pcdeditor{
		canvas: canvas,
		logPrint: func(msg interface{}) {
			fmt.Println(msg)
		},
		chImportPCD:         make(chan promiseCommand, 1),
		chImportSubPCD:      make(chan promiseCommand, 1),
		chImport2D:          make(chan promiseCommand, 1),
		chExportPCD:         make(chan promiseCommand, 1),
		chExportSelectedPCD: make(chan promiseCommand, 1),
		chReset:             make(chan promiseCommand, 1),
		chCommand:           make(chan promiseCommand, 1),
		chWheel:             make(chan webgl.WheelEvent, 10),
		chClick:             make(chan webgl.MouseEvent, 10),
		chMouseDown:         make(chan webgl.MouseEvent, 10),
		chMouseDrag:         make(chan webgl.MouseEvent, 10),
		chMouseMove:         make(chan webgl.MouseEvent, 10),
		chMouseUp:           make(chan webgl.MouseEvent, 10),
		chKey:               make(chan webgl.KeyboardEvent, 10),
		ch2D:                make(chan promiseCommand, 1),
		chContextLost:       make(chan webgl.WebGLContextEvent, 1),
		chContextRestored:   make(chan webgl.WebGLContextEvent, 1),

		vi:  newView(),
		cg:  &clickGuard{},
		cmd: newCommandContext(&pcdIOImpl{}, &mapIOImpl{}),
	}
	pe.cs = &console{cmd: pe.cmd, view: pe.vi}

	if len(args) > 1 {
		init := args[1]
		if logger := init.Get("logger"); !logger.IsUndefined() {
			pe.logPrint = func(msg interface{}) {
				logger.Invoke(fmt.Sprintf("%v", msg))
			}
		}
		if onKeyDownHook := init.Get("onKeyDownHook"); !onKeyDownHook.IsUndefined() {
			pe.onKeyDownHook = func(e webgl.KeyboardEvent) {
				onKeyDownHook.Invoke(e.JS())
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go pe.Run(ctx)

	return js.ValueOf(map[string]interface{}{
		"importPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chImportPCD, args[0])
		}),
		"importSubPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chImportSubPCD, args[0])
		}),
		"import2D": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chImport2D, [2]js.Value{args[0], args[1]})
		}),
		"exportPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chExportPCD, nil)
		}),
		"exportSelectedPCD": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chExportSelectedPCD, nil)
		}),
		"command": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chCommand, args[0].String())
		}),
		"show2D": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.ch2D, args[0].Bool())
		}),
		"reset": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return newCommandPromise(pe.chReset, nil)
		}),
		"exit": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			cancel()
			return nil
		}),
	})
}

func newCommandPromise(ch chan promiseCommand, data interface{}) js.Value {
	promise := js.Global().Get("Promise")
	return promise.New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve, reject := args[0], args[1]
		cmd := promiseCommand{
			data: data,
			resolved: func(res interface{}) {
				switch r := res.(type) {
				case [][]float32:
					jm := js.Global().Get("Array").New(len(r))
					for i, vec := range r {
						jv := js.Global().Get("Array").New(len(vec))
						for j, val := range vec {
							jv.SetIndex(j, js.ValueOf(val))
						}
						jm.SetIndex(i, jv)
					}
					resolve.Invoke(jm)
				default:
					resolve.Invoke(res)
				}
			},
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

func (pe *pcdeditor) Run(ctx context.Context) {
	canvas := webgl.Canvas(pe.canvas)

	canvas.OnClick(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chClick <- e:
		default:
		}
	})
	canvas.OnContextMenu(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
	})

	wheelHandler := func(e webgl.WheelEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chWheel <- e:
		default:
		}
	}
	canvas.OnWheel(wheelHandler)
	gesture := &gesture{
		canvas: canvas,
		onClick: func(e webgl.MouseEvent) {
			select {
			case pe.chClick <- e:
			default:
			}
		},
		onMouseDown: func(e webgl.MouseEvent) {
			select {
			case pe.chMouseDown <- e:
			default:
			}
		},
		onMouseDrag: func(e webgl.MouseEvent) {
			select {
			case pe.chMouseDrag <- e:
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
	canvas.OnTouchStart(gesture.touchStart)
	canvas.OnTouchMove(gesture.touchMove)
	canvas.OnTouchEnd(gesture.touchEnd)
	canvas.OnTouchCancel(gesture.touchEnd)

	mouseDragging := webgl.MouseButtonNull
	canvas.OnMouseUp(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chMouseUp <- e:
			if mouseDragging == e.Button {
				mouseDragging = webgl.MouseButtonNull
			}
		default:
		}
	})
	canvas.OnMouseDown(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chMouseDown <- e:
			mouseDragging = e.Button
		default:
		}
	})

	var lastMoveEvent *webgl.MouseEvent

	dispatchMoveEvent := func(e webgl.MouseEvent) {
		if e.Button == mouseDragging {
			select {
			case pe.chMouseDrag <- e:
			default:
			}
		} else {
			select {
			case pe.chMouseMove <- e:
			default:
			}
		}
	}
	canvas.OnMouseMove(func(e webgl.MouseEvent) {
		e.PreventDefault()
		e.StopPropagation()
		if mouseDragging != webgl.MouseButtonNull {
			e.Button = mouseDragging
		}
		lastMoveEvent = &e
		dispatchMoveEvent(e)
	})

	updateMove := func(e webgl.KeyboardEvent) {
		if lastMoveEvent != nil &&
			(e.ShiftKey != lastMoveEvent.ShiftKey ||
				e.CtrlKey != lastMoveEvent.CtrlKey ||
				e.AltKey != lastMoveEvent.AltKey) {
			// State of shift/ctrl/alt key is changed during drag
			lastMoveEvent.ShiftKey = e.ShiftKey
			lastMoveEvent.CtrlKey = e.CtrlKey
			lastMoveEvent.AltKey = e.AltKey
			dispatchMoveEvent(*lastMoveEvent)
		}
	}
	canvas.OnKeyDown(func(e webgl.KeyboardEvent) {
		if pe.onKeyDownHook != nil {
			pe.onKeyDownHook(e)
		}
		e.PreventDefault()
		e.StopPropagation()
		select {
		case pe.chKey <- e:
		default:
		}
		updateMove(e)
	})
	canvas.OnKeyUp(func(e webgl.KeyboardEvent) {
		e.PreventDefault()
		e.StopPropagation()
		updateMove(e)
	})

	canvas.OnWebGLContextLost(func(e webgl.WebGLContextEvent) {
		e.PreventDefault()
		e.StopPropagation()
		pe.chContextLost <- e
	})
	canvas.OnWebGLContextRestored(func(e webgl.WebGLContextEvent) {
		pe.chContextRestored <- e
	})

	for {
		err := pe.runImpl(ctx)
		switch err {
		case errContextLostEvent:
			// Received context lost event from the browser.
			pe.logPrint("Waiting WebGL context restore")
			<-pe.chContextRestored
			pe.logPrint("WebGL context restored")
		case errContextLost:
			// WebGL context is not available during initialization.
			time.Sleep(time.Second)
			pe.logPrint("Retrying")
		case nil:
			pe.logPrint("Exiting")
			println("pcdeditor exiting")
			return
		default:
			pe.logPrint("Fatal: " + err.Error())
			return
		}
	}
}

func (pe *pcdeditor) runImpl(ctx context.Context) error {
	gl, err := webgl.New(pe.canvas)
	if err != nil {
		return err
	}
	showDebugInfo(gl)

	vs, err := initVertexShader(gl, vsSource)
	if err != nil {
		return err
	}
	vsSub, err := initVertexShader(gl, vsSubSource)
	if err != nil {
		return err
	}
	vsSel, err := initVertexShader(gl, vsSelectSource)
	if err != nil {
		return err
	}
	vsMap, err := initVertexShader(gl, vsMapSource)
	if err != nil {
		return err
	}
	csComputeSelect, err := initVertexShader(gl, csComputeSelectSource)
	if err != nil {
		return err
	}
	fs, err := initFragmentShader(gl, fsSource)
	if err != nil {
		return err
	}
	fsMap, err := initFragmentShader(gl, fsMapSource)
	if err != nil {
		return err
	}
	fsComputeSelect, err := initFragmentShader(gl, fsComputeSelectSource)
	if err != nil {
		return err
	}

	program, err := linkShaders(gl, nil, vs, fs)
	if err != nil {
		return err
	}
	programSub, err := linkShaders(gl, nil, vsSub, fs)
	if err != nil {
		return err
	}
	programSel, err := linkShaders(gl, nil, vsSel, fs)
	if err != nil {
		return err
	}
	programMap, err := linkShaders(gl, nil, vsMap, fsMap)
	if err != nil {
		return err
	}
	programComputeSelect, err := linkShaders(gl, []string{"oResult"}, csComputeSelect, fsComputeSelect)
	if err != nil {
		return err
	}

	tf := gl.CreateTransformFeedback()
	gl.BindTransformFeedback(gl.TRANSFORM_FEEDBACK, tf)

	uProjectionMatrixLocation := gl.GetUniformLocation(program, "uProjectionMatrix")
	uCropMatrixLocation := gl.GetUniformLocation(program, "uCropMatrix")
	uModelViewMatrixLocation := gl.GetUniformLocation(program, "uModelViewMatrix")
	uSelectMatrixLocation := gl.GetUniformLocation(program, "uSelectMatrix")
	uZMinLocation := gl.GetUniformLocation(program, "uZMin")
	uZRangeLocation := gl.GetUniformLocation(program, "uZRange")
	uPointSizeBase := gl.GetUniformLocation(program, "uPointSizeBase")
	uUseSelectMask := gl.GetUniformLocation(program, "uUseSelectMask")

	uProjectionMatrixLocationSub := gl.GetUniformLocation(programSub, "uProjectionMatrix")
	uModelViewMatrixLocationSub := gl.GetUniformLocation(programSub, "uModelViewMatrix")
	uPointSizeBaseSub := gl.GetUniformLocation(programSub, "uPointSizeBase")

	uProjectionMatrixLocationSel := gl.GetUniformLocation(programSel, "uProjectionMatrix")
	uModelViewMatrixLocationSel := gl.GetUniformLocation(programSel, "uModelViewMatrix")
	uPointSizeBaseSel := gl.GetUniformLocation(programSel, "uPointSizeBase")

	uProjectionMatrixLocationMap := gl.GetUniformLocation(programMap, "uProjectionMatrix")
	uModelViewMatrixLocationMap := gl.GetUniformLocation(programMap, "uModelViewMatrix")
	uSamplerLocationMap := gl.GetUniformLocation(programMap, "uSampler")
	uAlphaLocationMap := gl.GetUniformLocation(programMap, "uAlpha")

	uCropMatrixLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uCropMatrix")
	uSelectMatrixLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uSelectMatrix")
	uProjectionMatrixLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uProjectionMatrix")
	uModelViewMatrixLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uModelViewMatrix")
	uOriginLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uOrigin")
	uDirLocationComputeSelect := gl.GetUniformLocation(programComputeSelect, "uDir")

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	posBuf := gl.CreateBuffer()
	posSubBuf := gl.CreateBuffer()
	mapBuf := gl.CreateBuffer()
	selectResultBuf := gl.CreateBuffer()
	selectMaskBuf := gl.CreateBuffer()
	toolBuf := gl.CreateBuffer()
	var selectResultJS js.Value
	var selectResultGo []byte

	tick := time.NewTicker(time.Second / 8)
	defer tick.Stop()

	var fov float32
	var projectionMatrix, modelViewMatrix mat.Mat4
	var width, height int
	var distance float64
	var projectionType ProjectionType

	var vib3D bool
	var vib3DX float32

	var nRectPoints int

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)

	const (
		aVertexPosition  = 0
		aVertexLabel     = 1
		aTextureCoordMap = 1
		aSelectMask      = 2
	)

	devicePixelRatioJS := js.Global().Get("window").Get("devicePixelRatio")
	wheelNormalizer := &wheelNormalizer{}

	texture := gl.CreateTexture()
	mapRect := &pc.PointCloud{
		PointCloudHeader: pc.PointCloudHeader{
			Fields: []string{"x", "y", "z", "u", "v"},
			Size:   []int{4, 4, 4, 4, 4},
			Count:  []int{1, 1, 1, 1, 1},
		},
		Points: 5,
		Data:   make([]byte, 5*4*5),
	}
	var selectMaskData webgl.ByteArrayBuffer
	var pcCursor *pc.PointCloud
	var moveStart *mat.Vec3
	var show2D bool = true

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
			if pp, _, ok := pe.cmd.PointCloud(); ok {
				pe.logPrint("CRASHED (export command is available)")
				pe.logPrint("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				for promise := range pe.chExportPCD {
					blob, err := pe.cmd.pcdIO.exportPCD(pp)
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

	forceReload := true

	pe.logPrint("WebGL context initialized")

	for {
		scale := float32(devicePixelRatioJS.Float())
		scaled := func(v int) int {
			return int(float32(v) * scale)
		}
		newWidth := scaled(gl.Canvas.ClientWidth())
		newHeight := scaled(gl.Canvas.ClientHeight())
		newProjectionType := pe.cmd.ProjectionType()
		newDistance := pe.vi.distance
		newFOV := pe.vi.fov

		if forceReload || newWidth != width || newHeight != height || newFOV != fov || projectionType != newProjectionType || (newProjectionType == ProjectionOrthographic && newDistance != distance) {
			width, height = newWidth, newHeight
			projectionType = newProjectionType
			distance = newDistance
			fov = newFOV

			gl.Canvas.SetWidth(width)
			gl.Canvas.SetHeight(height)
			switch projectionType {
			case ProjectionPerspective:
				projectionMatrix = mat.Perspective(
					fov,
					float32(width)/float32(height),
					1.0, 1000.0,
				)
			case ProjectionOrthographic:
				projectionMatrix = mat.Orthographic(
					-float32(width/2)*float32(distance)/1000,
					float32(width/2)*float32(distance)/1000,
					float32(height/2)*float32(distance)/1000,
					-float32(height/2)*float32(distance)/1000,
					-1000, 1000.0,
				)
			}
			gl.UseProgram(program)
			gl.UniformMatrix4fv(uProjectionMatrixLocation, false, projectionMatrix)
			gl.UseProgram(programSub)
			gl.UniformMatrix4fv(uProjectionMatrixLocationSub, false, projectionMatrix)
			gl.UseProgram(programSel)
			gl.UniformMatrix4fv(uProjectionMatrixLocationSel, false, projectionMatrix)
			gl.UseProgram(programMap)
			gl.UniformMatrix4fv(uProjectionMatrixLocationMap, false, projectionMatrix)
			gl.UseProgram(programComputeSelect)
			gl.UniformMatrix4fv(uProjectionMatrixLocationComputeSelect, false, projectionMatrix)
			gl.Viewport(0, 0, width, height)
		}

		modelViewMatrix = mat.Rotate(1, 0, 0, -float32(pe.vi.pitch)).
			MulAffine(mat.Rotate(0, 0, 1, -float32(pe.vi.yaw))).
			MulAffine(mat.Translate(float32(pe.vi.x), float32(pe.vi.y), -1.5))
		if projectionType == ProjectionPerspective {
			modelViewMatrix =
				mat.Translate(vib3DX, 0, -float32(pe.vi.distance)).MulAffine(modelViewMatrix)
		}

		if rect, updated := pe.cmd.Rect(); updated || forceReload {
			// Send select box vertices to GPU
			buf := make([]float32, 0, len(rect)*3)
			for _, p := range rect {
				buf = append(buf, p[0], p[1], p[2])
			}
			nRectPoints = len(rect)
			if len(rect) > 0 {
				pcCursor = &pc.PointCloud{
					PointCloudHeader: pc.PointCloudHeader{
						Fields: []string{"x", "y", "z"},
						Size:   []int{4, 4, 4},
						Type:   []string{"F", "F", "F"},
						Count:  []int{1, 1, 1},
						Width:  nRectPoints,
						Height: 1,
					},
					Points: nRectPoints,
				}
				pcCursor.Data = make([]byte, nRectPoints*pcCursor.Stride())
				it, _ := pcCursor.Vec3Iterator()
				for _, p := range rect {
					it.SetVec3(p)
					it.Incr()
				}

				gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
				gl.BufferData(gl.ARRAY_BUFFER, webgl.Float32ArrayBuffer(buf), gl.STATIC_DRAW)
			}
		}

		mi, img, mapUpdated, has2D := pe.cmd.Map()
		if has2D && (mapUpdated || forceReload) {
			// Send 2D map texture to GPU
			err0 := gl.GetError()
			gl.BindTexture(gl.TEXTURE_2D, texture)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img.Interface().(js.Value))
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
			gl.BindTexture(gl.TEXTURE_2D, webgl.Texture(nil))

			if err := gl.GetError(); err0 == nil && err != nil {
				pe.logPrint(fmt.Sprintf("Failed to render 2D map image (%v): 2D map image size may be too large for your graphic card", err))
			}

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

		updateSelectMask := func() {
			gl.UseProgram(program)
			gl.BindBuffer(gl.ARRAY_BUFFER, selectMaskBuf)
			gl.BufferData(gl.ARRAY_BUFFER, selectMaskData, gl.STATIC_DRAW)
		}

		pp, updatedPointCloud, hasPointCloud := pe.cmd.PointCloud()
		if hasPointCloud && (updatedPointCloud || forceReload) && pp.Points > 0 {
			// Send PointCloud vertices to GPU
			gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
			gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(pp.Data), gl.STATIC_DRAW)

			// Re-allocate buffer only when pointcloud size is changed
			if nBuf := pp.Points * 4; nBuf != len(selectResultGo) {
				// Register buffer to receive GPGPU processing result
				selectResultJS = js.Global().Get("Uint8Array").New(nBuf)
				if cap(selectResultGo) < nBuf {
					selectResultGo = make([]byte, nBuf)
				}
				selectResultGo = selectResultGo[:nBuf:nBuf]
				selectMaskData = webgl.ByteArrayBuffer(selectResultGo)
			}
			gl.BindBuffer(gl.ARRAY_BUFFER, selectResultBuf)
			gl.BufferData_JS(gl.ARRAY_BUFFER, js.ValueOf(selectResultJS), gl.STREAM_READ)

			updateSelectMask()
		}

		ppSub, updatedSubPointCloud, hasSubPointCloud := pe.cmd.SubPointCloud()
		if hasSubPointCloud && (updatedSubPointCloud || forceReload) && ppSub.Points > 0 {
			// Send PointCloud vertices to GPU
			gl.BindBuffer(gl.ARRAY_BUFFER, posSubBuf)
			gl.BufferData(gl.ARRAY_BUFFER, webgl.ByteArrayBuffer(ppSub.Data), gl.STATIC_DRAW)
		}

		render := func() {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

			pointSize := pe.cmd.PointSize()
			selectMode := pe.cmd.SelectMode()

			if hasPointCloud && pp.Points > 0 {
				// Render PointCloud
				gl.UseProgram(program)
				clean := enableVertexAttribs(gl, aVertexPosition, aVertexLabel, aSelectMask)

				switch selectMode {
				case selectModeRect, selectModeInsert:
					gl.Uniform1i(uUseSelectMask, 0)
				case selectModeMask:
					gl.Uniform1i(uUseSelectMask, 1)
				}

				gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
				gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, pp.Stride(), 0)
				gl.VertexAttribIPointer(aVertexLabel, 1, gl.UNSIGNED_INT, pp.Stride(), 3*4)
				gl.UniformMatrix4fv(uModelViewMatrixLocation, false, modelViewMatrix)
				gl.UniformMatrix4fv(uCropMatrixLocation, false, pe.cmd.CropMatrix())

				gl.BindBuffer(gl.ARRAY_BUFFER, selectMaskBuf)
				gl.VertexAttribIPointer(aSelectMask, 1, gl.UNSIGNED_INT, 4, 0)

				zMin, zMax := pe.cmd.ZRange()
				gl.Uniform1f(uZMinLocation, zMin)
				gl.Uniform1f(uZRangeLocation, zMax-zMin)

				mSel, _ := pe.cmd.SelectMatrix()
				gl.UniformMatrix4fv(uSelectMatrixLocation, false, mSel)

				gl.Uniform1f(uPointSizeBase, pointSize)

				gl.DrawArrays(gl.POINTS, 0, pp.Points-1)
				clean()
			}

			if hasSubPointCloud && ppSub.Points > 0 && selectMode == selectModeInsert {
				// Render sub PointCloud
				cursors := pe.cmd.Cursors()

				gl.Enable(gl.BLEND)
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				gl.UseProgram(programSub)
				clean := enableVertexAttribs(gl, aVertexPosition)
				gl.BindBuffer(gl.ARRAY_BUFFER, posSubBuf)
				gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, ppSub.Stride(), 0)
				trans := cursorsToTrans(cursors)
				gl.UniformMatrix4fv(
					uModelViewMatrixLocationSub, false,
					modelViewMatrix.Mul(trans),
				)
				gl.Uniform1f(uPointSizeBaseSub, pointSize)
				gl.DrawArrays(gl.POINTS, 0, ppSub.Points-1)

				gl.Disable(gl.BLEND)
				clean()
			}

			if nRectPoints > 0 && (selectMode == selectModeRect || selectMode == selectModeInsert) {
				// Render select box
				gl.Enable(gl.BLEND)
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				gl.UseProgram(programSel)
				clean := enableVertexAttribs(gl, aVertexPosition)
				for i := 0; i < nRectPoints; i += 4 {
					gl.BindBuffer(gl.ARRAY_BUFFER, toolBuf)
					gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, 3*4, 3*4*i)
					n := 4
					if n > nRectPoints-i {
						n = nRectPoints - i
					}

					gl.UniformMatrix4fv(uModelViewMatrixLocationSel, false, modelViewMatrix)
					gl.Uniform1f(uPointSizeBaseSel, pointSize)
					gl.DrawArrays(gl.LINE_LOOP, 0, n)
					gl.DrawArrays(gl.POINTS, 0, n)
				}
				gl.Disable(gl.BLEND)
				clean()
			}

			if show2D && has2D {
				// Render 2D map
				gl.Enable(gl.BLEND)
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				gl.UseProgram(programMap)
				clean := enableVertexAttribs(gl, aVertexPosition, aTextureCoordMap)
				gl.BindBuffer(gl.ARRAY_BUFFER, mapBuf)
				gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, mapRect.Stride(), 0)
				gl.VertexAttribPointer(aTextureCoordMap, 2, gl.FLOAT, false, mapRect.Stride(), 4*3)
				gl.UniformMatrix4fv(uModelViewMatrixLocationMap, false, modelViewMatrix)
				gl.Uniform1f(uAlphaLocationMap, pe.cmd.MapAlpha())
				gl.DrawArrays(gl.TRIANGLE_FAN, 0, 5)
				gl.Disable(gl.BLEND)
				clean()
			}
		}
		render()

		// Calculate condition of each point by GPU
		// It checks that the point is
		//   - in the crop box
		//   - in the select box
		//   - close to the mouse cursor position given as (x, y)
		scanSelection := func(x, y int) bool {
			if hasPointCloud && pp.Points > 0 {
				origin, dir := perspectiveOriginDir(x, y, width, height, &projectionMatrix, &modelViewMatrix)

				// Run GPGPU shader
				gl.UseProgram(programComputeSelect)
				clean := enableVertexAttribs(gl, aVertexPosition, aSelectMask)
				defer clean()

				gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
				gl.VertexAttribPointer(aVertexPosition, 3, gl.FLOAT, false, pp.Stride(), 0)
				gl.BindBuffer(gl.ARRAY_BUFFER, selectMaskBuf)
				gl.VertexAttribIPointer(aSelectMask, 1, gl.UNSIGNED_INT, 4, 0)

				gl.UniformMatrix4fv(uCropMatrixLocationComputeSelect, false, pe.cmd.CropMatrix())
				gl.UniformMatrix4fv(uModelViewMatrixLocationComputeSelect, false, modelViewMatrix)

				mSel, _ := pe.cmd.SelectMatrix()
				gl.UniformMatrix4fv(uSelectMatrixLocationComputeSelect, false, mSel)

				gl.Uniform3fv(uOriginLocationComputeSelect, *origin)
				gl.Uniform3fv(uDirLocationComputeSelect, *dir)

				gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, selectResultBuf)
				gl.Enable(gl.RASTERIZER_DISCARD)
				gl.BeginTransformFeedback(gl.POINTS)
				gl.DrawArrays(gl.POINTS, 0, pp.Points-1)
				gl.EndTransformFeedback()
				gl.Disable(gl.RASTERIZER_DISCARD)
				gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, webgl.Buffer(js.Null()))

				fence := gl.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
				defer func() {
					gl.DeleteSync(fence)
				}()

				// Re-render to avoid blank screen on Firefox
				render()

				// Switch execution frame first to ensure state update
				time.Sleep(time.Millisecond)

				// Wait calculation on GPU
			L_SYNC:
				for failCnt := 0; ; {
					switch gl.ClientWaitSync(fence, 0, 0) {
					case gl.ALREADY_SIGNALED, gl.CONDITION_SATISFIED:
						break L_SYNC
					case gl.WAIT_FAILED:
						if failCnt++; failCnt > 10 {
							return false
						}
					}
					time.Sleep(10 * time.Millisecond)
				}

				// Get result from GPU
				gl.BindBuffer(gl.ARRAY_BUFFER, selectResultBuf)
				gl.GetBufferSubData(gl.ARRAY_BUFFER, 0, selectResultJS, 0, 0)
				js.CopyBytesToGo(selectResultGo, selectResultJS)

				pe.cmd.SetSelectMask(webgl.ByteArrayBuffer(selectResultGo).UInt32Slice())
				return true
			}
			return false
		}

		// Check the cursor is on select box vertices
		cursorOnSelect := func(e webgl.MouseEvent) (*mat.Vec3, bool) {
			if nRectPoints == 0 || pcCursor == nil {
				return nil, false
			}
			return selectPoint(
				pcCursor, nil, projectionType, &modelViewMatrix, &projectionMatrix,
				scaled(e.OffsetX), scaled(e.OffsetY), width, height, rectSelectRange,
			)
		}

		forceReload = false

		// Handle inputs
		select {
		case promise := <-pe.chImportPCD:
			pe.logPrint("importing pcd")
			if err := pe.cmd.ImportPCD(promise.data); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd loaded")
			promise.resolved("loaded")
		case promise := <-pe.chImportSubPCD:
			pe.logPrint("importing sub pcd")
			if err := pe.cmd.ImportSubPCD(promise.data); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("sub pcd loaded")
			promise.resolved("loaded")
		case promise := <-pe.chImport2D:
			pe.logPrint("loading 2D map")
			data := promise.data.([2]js.Value)
			if err := pe.cmd.Import2D(data[0], data[1]); err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("2D map loaded")
			promise.resolved("loaded")
		case promise := <-pe.chExportPCD:
			pe.logPrint("exporting pcd")
			blob, err := pe.cmd.ExportPCD()
			if err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd exported")
			promise.resolved(blob)
		case promise := <-pe.chExportSelectedPCD:
			pe.logPrint("exporting selected points as pcd")
			if !scanSelection(0, 0) {
				promise.rejected(errors.New("failed to scan selected points"))
			}
			blob, err := pe.cmd.ExportSelectedPCD()
			if err != nil {
				promise.rejected(err)
				break
			}
			pe.logPrint("pcd exported")
			promise.resolved(blob)
		case promise := <-pe.chReset:
			pe.cmd.Reset()
			promise.resolved("resetted")
		case promise := <-pe.chCommand:
			res, err := pe.cs.Run(promise.data.(string), func() error {
				if scanSelection(0, 0) {
					return nil
				}
				return errors.New("failed to scan selected points")
			})
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
				if len(pe.cmd.Cursors()) < 4 {
					pe.cmd.SetSelectRange(
						rangeTypeAuto,
						pe.cmd.SelectRange(rangeTypeAuto)+float32(e.DeltaY*rate),
					)
					break
				}
				r := 1.0 + float32(e.DeltaY*rate)
				m, _ := pe.cmd.SelectMatrix()
				pe.cmd.TransformCursors(
					m.InvAffine().
						MulAffine(mat.Translate(0, 0, 0.5)).
						MulAffine(mat.Scale(1, 1, r)).
						MulAffine(mat.Translate(0, 0, -0.5)).
						MulAffine(m),
				)
			case e.ShiftKey:
				rect, _ := pe.cmd.Rect()
				if len(rect) > 0 {
					var c mat.Vec3
					for _, p := range rect {
						c = c.Add(p)
					}
					c = c.Mul(1.0 / float32(len(rect)))
					r := 1.0 + float32(e.DeltaY)*0.01
					pe.cmd.TransformCursors(
						mat.Translate(c[0], c[1], c[2]).
							MulAffine(mat.Scale(r, r, r)).
							MulAffine(mat.Translate(-c[0], -c[1], -c[2])),
					)
				}
			default:
				pe.vi.wheel(&e)
			}
		case e := <-pe.chMouseDown:
			gl.Canvas.Focus()
			if e.Button == 0 {
				pe.cg.DragStart()
				if p, ok := cursorOnSelect(e); ok {
					pe.cmd.PushCursors()
					moveStart = selectPointOrtho(
						&modelViewMatrix, &projectionMatrix,
						scaled(e.OffsetX), scaled(e.OffsetY), width, height, p,
					)
					continue
				}
			}
			moveStart = nil
			pe.vi.mouseDragStart(&e)
		case e := <-pe.chMouseUp:
			if e.Button == 0 {
				pe.cg.DragEnd()
			}
			if moveStart != nil {
				pe.cmd.PopCursors()
				moveEnd := selectPointOrtho(
					&modelViewMatrix, &projectionMatrix,
					scaled(e.OffsetX), scaled(e.OffsetY), width, height, moveStart,
				)
				var trans mat.Mat4
				switch {
				case e.ShiftKey:
					rect, _ := pe.cmd.Rect()
					trans = dragRotation(*moveStart, *moveEnd, rect, &modelViewMatrix)
				default:
					trans = dragTranslation(*moveStart, *moveEnd)
				}
				pe.cmd.TransformCursors(trans)
				moveStart = nil
				continue
			}
			pe.vi.mouseDragEnd(&e)
		case e := <-pe.chMouseDrag:
			pe.cg.Move()
			if e.Button == 0 && moveStart != nil {
				pe.cmd.PopCursors()
				pe.cmd.PushCursors()

				if e.ShiftKey {
					pe.SetCursor(cursorGrabbing)
				} else {
					pe.SetCursor(cursorMove)
				}

				moveEnd := selectPointOrtho(
					&modelViewMatrix, &projectionMatrix,
					scaled(e.OffsetX), scaled(e.OffsetY), width, height, moveStart,
				)
				var trans mat.Mat4
				switch {
				case e.ShiftKey:
					rect, _ := pe.cmd.Rect()
					trans = dragRotation(*moveStart, *moveEnd, rect, &modelViewMatrix)
				default:
					trans = dragTranslation(*moveStart, *moveEnd)
				}
				pe.cmd.TransformCursors(trans)
				continue
			}
			pe.vi.mouseDrag(&e)
		case e := <-pe.chMouseMove:
			if _, ok := cursorOnSelect(e); ok {
				if e.ShiftKey {
					pe.SetCursor(cursorGrab)
				} else {
					pe.SetCursor(cursorMove)
				}
			} else {
				pe.SetCursor(cursorAuto)
			}
		case e := <-pe.chClick:
			gl.Canvas.Focus()
			if e.Button != 0 || !pe.cg.Click() {
				continue
			}
			ok := scanSelection(scaled(e.OffsetX), scaled(e.OffsetY))
			if !ok {
				updateSelectMask()
				continue
			}
			var p *mat.Vec3
			switch projectionType {
			case ProjectionPerspective:
				p, ok = selectPoint(
					pp, pe.cmd.SelectMask(), projectionType, &modelViewMatrix, &projectionMatrix,
					scaled(e.OffsetX), scaled(e.OffsetY), width, height, pointSelectRange,
				)
			case ProjectionOrthographic:
				p = selectPointOrtho(
					&modelViewMatrix, &projectionMatrix, scaled(e.OffsetX), scaled(e.OffsetY), width, height, nil,
				)
			default:
				ok = false
			}
			if !ok {
				updateSelectMask()
				continue
			}
			switch {
			case e.ShiftKey:
				if len(pe.cmd.Cursors()) < 3 {
					pe.cmd.SetCursor(1, *p)
				} else {
					pe.cmd.SetCursor(3, *p)
				}
			case e.AltKey:
				if projectionType != ProjectionPerspective {
					break
				}
				if ok := scanSelection(scaled(e.OffsetX), scaled(e.OffsetY)); ok {
					pe.cmd.SelectSegment(*p)
					updateSelectMask()
				}
			case e.CtrlKey:
				if projectionType != ProjectionPerspective {
					break
				}
				if ok := scanSelection(scaled(e.OffsetX), scaled(e.OffsetY)); ok {
					err := pe.cmd.SelectLabelSegment(*p)
					if err != nil {
						pe.logPrint("Selection by label failed: " + err.Error())
					} else {
						updateSelectMask()
					}
				}
			default:
				if len(pe.cmd.Cursors()) < 2 {
					pe.cmd.SetCursor(0, *p)
				} else {
					pe.cmd.SetCursor(2, *p)
				}
			}
		case e := <-pe.chKey:
			switch e.Code {
			case "Escape":
				if moveStart != nil {
					pe.cmd.PopCursors()
					moveStart = nil
				} else {
					pe.cmd.UnsetCursors()
				}
			case "Enter":
				if err := pe.cmd.FinalizeCurrentMode(); err != nil {
					pe.logPrint("Failed: " + err.Error())
				}
			case "Delete", "Backspace", "Digit0", "Digit1":
				switch e.Code {
				case "Delete", "Backspace":
					if ok := scanSelection(0, 0); ok {
						pe.cmd.Delete()
						if !e.ShiftKey && !e.CtrlKey {
							pe.cmd.UnsetCursors()
						}
					}
				case "Digit0", "Digit1":
					var l uint32
					if e.Code == "Digit1" {
						l = 1
					}
					if ok := scanSelection(0, 0); ok {
						pe.cmd.Label(l)
					}
				}
			case "KeyZ":
				if e.CtrlKey {
					pe.cmd.Undo()
				}
			case "KeyU":
				pe.cmd.Undo()
			case "KeyF":
				pe.cmd.AddSurface(defaultResolution)
			case "KeyV", "KeyH":
				switch e.Code {
				case "KeyV":
					pe.cmd.SnapVertical()
				case "KeyH":
					pe.cmd.SnapHorizontal()
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
				s, c := math.Sincos(pe.vi.yaw)
				pe.cmd.TransformCursors(mat.Translate(
					float32(c)*dx-float32(s)*dy,
					float32(s)*dx+float32(c)*dy,
					dz,
				))
			case "Home", "End":
				var dyaw float32
				switch e.Code {
				case "Home":
					dyaw = 0.005
				case "End":
					dyaw = -0.005
				}
				center := pe.cmd.RectCenterPos()
				pe.cmd.TransformCursors(
					mat.Translate(center[0], center[1], center[2]).
						Mul(mat.Rotate(0, 0, 1, dyaw)).
						Mul(mat.Translate(-center[0], -center[1], -center[2])),
				)
			case "KeyW", "KeyA", "KeyS", "KeyD", "KeyQ", "KeyE":
				switch e.Code {
				case "KeyW":
					pe.vi.Move(0.05, 0, 0)
				case "KeyA":
					pe.vi.Move(0, 0.05, 0)
				case "KeyS":
					pe.vi.Move(-0.05, 0, 0)
				case "KeyD":
					pe.vi.Move(0, -0.05, 0)
				case "KeyQ":
					pe.vi.Move(0, 0, 0.02)
				case "KeyE":
					pe.vi.Move(0, 0, -0.02)
				}
			case "BracketRight", "Backslash":
				switch e.Code {
				case "BracketRight":
					pe.vi.IncreaseFOV()
				case "Backslash":
					pe.vi.DecreaseFOV()
				}
			case "F10":
				pe.cmd.Crop()
			case "F11":
				pe.vi.SnapYaw()
			case "F12":
				pe.vi.SnapPitch()
			case "KeyP":
				vib3D = !vib3D
			}
		case <-pe.chContextLost:
			return errContextLostEvent
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
		case <-ctx.Done():
			return nil
		}
	}
}
