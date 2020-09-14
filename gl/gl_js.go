package gl

import (
	"errors"
	"syscall/js"

	"github.com/seqsense/pcdeditor/mat"
)

var (
	float32Array = js.Global().Get("Float32Array")
	uint8Array   = js.Global().Get("Uint8Array")
)

type ShaderType int
type BufferType int
type BufferUsage int
type Capacity int
type DepthFunc int
type Type int
type BufferMask int
type DrawMode int
type ProgramParameter int
type ShaderParameter int
type TextureType int
type PixelFormat int
type TextureParameter int
type TextureNumber int
type BlendFactor int

type Shader js.Value
type Program js.Value
type Location js.Value
type Buffer js.Value
type Texture *js.Value

type WebGL struct {
	gl js.Value

	Canvas Canvas

	VERTEX_SHADER, FRAGMENT_SHADER                                                ShaderType
	ARRAY_BUFFER                                                                  BufferType
	STATIC_DRAW                                                                   BufferUsage
	DEPTH_TEST, BLEND                                                             Capacity
	LEQUAL                                                                        DepthFunc
	FLOAT, UNSIGNED_BYTE, UNSIGNED_SHORT, UNSIGNED_INT                            Type
	COLOR_BUFFER_BIT, DEPTH_BUFFER_BIT, STENCIL_BUFFER_BIT                        BufferMask
	POINTS, LINE_STRIP, LINE_LOOP, LINES, TRIANGLE_STRIP, TRIANGLE_FAN, TRIANGLES DrawMode
	COMPILE_STATUS                                                                ShaderParameter
	LINK_STATUS, VALIDATE_STATUS                                                  ProgramParameter
	TEXTURE_2D                                                                    TextureType

	RGBA PixelFormat

	TEXTURE_MIN_FILTER, TEXTURE_WRAP_S, TEXTURE_WRAP_T TextureParameter
	LINEAR, NEAREST, CLAMP_TO_EDGE                     int

	TEXTURE0 TextureNumber

	ZERO, ONE, SRC_ALPHA, ONE_MINUS_SRC_ALPHA BlendFactor
}

func New(canvas js.Value) (*WebGL, error) {
	gl := canvas.Call("getContext", "webgl2")
	if gl.IsNull() {
		return nil, errors.New("WebGL is not supported")
	}
	return &WebGL{
		gl: gl,

		Canvas: Canvas(gl.Get("canvas")),

		VERTEX_SHADER:   ShaderType(gl.Get("VERTEX_SHADER").Int()),
		FRAGMENT_SHADER: ShaderType(gl.Get("FRAGMENT_SHADER").Int()),

		ARRAY_BUFFER: BufferType(gl.Get("ARRAY_BUFFER").Int()),

		STATIC_DRAW: BufferUsage(gl.Get("STATIC_DRAW").Int()),

		DEPTH_TEST: Capacity(gl.Get("DEPTH_TEST").Int()),
		BLEND:      Capacity(gl.Get("BLEND").Int()),

		LEQUAL: DepthFunc(gl.Get("LEQUAL").Int()),

		FLOAT:          Type(gl.Get("FLOAT").Int()),
		UNSIGNED_BYTE:  Type(gl.Get("UNSIGNED_BYTE").Int()),
		UNSIGNED_SHORT: Type(gl.Get("UNSIGNED_SHORT").Int()),
		UNSIGNED_INT:   Type(gl.Get("UNSIGNED_INT").Int()),

		COLOR_BUFFER_BIT:   BufferMask(gl.Get("COLOR_BUFFER_BIT").Int()),
		DEPTH_BUFFER_BIT:   BufferMask(gl.Get("DEPTH_BUFFER_BIT").Int()),
		STENCIL_BUFFER_BIT: BufferMask(gl.Get("STENCIL_BUFFER_BIT").Int()),

		POINTS:         DrawMode(gl.Get("POINTS").Int()),
		LINE_STRIP:     DrawMode(gl.Get("LINE_STRIP").Int()),
		LINE_LOOP:      DrawMode(gl.Get("LINE_LOOP").Int()),
		LINES:          DrawMode(gl.Get("LINES").Int()),
		TRIANGLE_STRIP: DrawMode(gl.Get("TRIANGLE_STRIP").Int()),
		TRIANGLE_FAN:   DrawMode(gl.Get("TRIANGLE_FAN").Int()),
		TRIANGLES:      DrawMode(gl.Get("TRIANGLES").Int()),

		COMPILE_STATUS: ShaderParameter(gl.Get("COMPILE_STATUS").Int()),

		LINK_STATUS:     ProgramParameter(gl.Get("LINK_STATUS").Int()),
		VALIDATE_STATUS: ProgramParameter(gl.Get("VALIDATE_STATUS").Int()),

		TEXTURE_2D: TextureType(gl.Get("TEXTURE_2D").Int()),

		RGBA: PixelFormat(gl.Get("RGBA").Int()),

		TEXTURE_MIN_FILTER: TextureParameter(gl.Get("TEXTURE_MIN_FILTER").Int()),
		TEXTURE_WRAP_S:     TextureParameter(gl.Get("TEXTURE_WRAP_S").Int()),
		TEXTURE_WRAP_T:     TextureParameter(gl.Get("TEXTURE_WRAP_T").Int()),

		LINEAR:        gl.Get("LINEAR").Int(),
		NEAREST:       gl.Get("NEAREST").Int(),
		CLAMP_TO_EDGE: gl.Get("CLAMP_TO_EDGE").Int(),

		TEXTURE0: TextureNumber(gl.Get("TEXTURE0").Int()),

		ZERO:                BlendFactor(gl.Get("ZERO").Int()),
		ONE:                 BlendFactor(gl.Get("ONE").Int()),
		SRC_ALPHA:           BlendFactor(gl.Get("SRC_ALPHA").Int()),
		ONE_MINUS_SRC_ALPHA: BlendFactor(gl.Get("ONE_MINUS_SRC_ALPHA").Int()),
	}, nil
}

func (gl *WebGL) CreateShader(t ShaderType) Shader {
	return Shader(gl.gl.Call("createShader", int(t)))
}

func (gl *WebGL) ShaderSource(s Shader, src string) {
	gl.gl.Call("shaderSource", js.Value(s), src)
}

func (gl *WebGL) CompileShader(s Shader) {
	gl.gl.Call("compileShader", js.Value(s))
}

func (gl *WebGL) GetShaderParameter(s Shader, param ShaderParameter) interface{} {
	v := gl.gl.Call("getShaderParameter", js.Value(s), int(param))
	switch param {
	case gl.COMPILE_STATUS:
		return v.Bool()
	}
	return nil
}

func (gl *WebGL) CreateProgram() Program {
	return Program(gl.gl.Call("createProgram"))
}

func (gl *WebGL) AttachShader(p Program, s Shader) {
	gl.gl.Call("attachShader", js.Value(p), js.Value(s))
}

func (gl *WebGL) LinkProgram(p Program) {
	gl.gl.Call("linkProgram", js.Value(p))
}

func (gl *WebGL) GetProgramParameter(p Program, param ProgramParameter) interface{} {
	v := gl.gl.Call("getProgramParameter", js.Value(p), int(param))
	switch param {
	case gl.LINK_STATUS, gl.VALIDATE_STATUS:
		return v.Bool()
	}
	return nil
}

func (gl *WebGL) GetProgramInfoLog(p Program) string {
	return gl.gl.Call("getProgramInfoLog", js.Value(p)).String()
}

func (gl *WebGL) UseProgram(p Program) {
	gl.gl.Call("useProgram", js.Value(p))
}

func (gl *WebGL) GetAttribLocation(p Program, name string) int {
	return gl.gl.Call("getAttribLocation", js.Value(p), name).Int()
}

func (gl *WebGL) GetUniformLocation(p Program, name string) Location {
	return Location(gl.gl.Call("getUniformLocation", js.Value(p), name))
}

func (gl *WebGL) CreateBuffer() Buffer {
	return Buffer(gl.gl.Call("createBuffer"))
}

func (gl *WebGL) BindBuffer(t BufferType, buf Buffer) {
	gl.gl.Call("bindBuffer", int(t), js.Value(buf))
}

func (gl *WebGL) BufferData(t BufferType, data BufferData, usage BufferUsage) {
	bin := data.Bytes()
	dataJS := uint8Array.New(len(bin))
	js.CopyBytesToJS(dataJS, bin)
	gl.gl.Call("bufferData", int(t), dataJS, int(usage))
}

func (gl *WebGL) ClearColor(r, g, b, a float32) {
	gl.gl.Call("clearColor", r, g, b, a)
}

func (gl *WebGL) ClearDepth(d float32) {
	gl.gl.Call("clearDepth", d)
}

func (gl *WebGL) Enable(c Capacity) {
	gl.gl.Call("enable", int(c))
}

func (gl *WebGL) Disable(c Capacity) {
	gl.gl.Call("disable", int(c))
}

func (gl *WebGL) DepthFunc(f DepthFunc) {
	gl.gl.Call("depthFunc", int(f))
}

func (gl *WebGL) VertexAttribPointer(i, size int, typ Type, normalized bool, stride, offset int) {
	gl.gl.Call("vertexAttribPointer", i, size, int(typ), normalized, stride, offset)
}

func (gl *WebGL) VertexAttribIPointer(i, size int, typ Type, stride, offset int) {
	gl.gl.Call("vertexAttribIPointer", i, size, int(typ), stride, offset)
}

func (gl *WebGL) EnableVertexAttribArray(i int) {
	gl.gl.Call("enableVertexAttribArray", i)
}

func (gl *WebGL) UniformMatrix4fv(loc Location, transpose bool, mat mat.Mat4) {
	matJS := float32Array.Call("of",
		mat[0], mat[1], mat[2], mat[3],
		mat[4], mat[5], mat[6], mat[7],
		mat[8], mat[9], mat[10], mat[11],
		mat[12], mat[13], mat[14], mat[15],
	)
	gl.gl.Call("uniformMatrix4fv", js.Value(loc), transpose, matJS)
}

func (gl *WebGL) Uniform3fv(loc Location, v mat.Vec3) {
	vecJS := float32Array.Call("of", v[0], v[1], v[2])
	gl.gl.Call("uniform3fv", js.Value(loc), vecJS)
}

func (gl *WebGL) Uniform1i(loc Location, i int) {
	gl.gl.Call("uniform1i", js.Value(loc), i)
}

func (gl *WebGL) Uniform1f(loc Location, i float32) {
	gl.gl.Call("uniform1f", js.Value(loc), i)
}

func (gl *WebGL) Clear(mask BufferMask) {
	gl.gl.Call("clear", int(mask))
}

func (gl *WebGL) DrawArrays(mode DrawMode, i, n int) {
	gl.gl.Call("drawArrays", int(mode), i, n)
}

func (gl *WebGL) Viewport(x1, y1, x2, y2 int) {
	gl.gl.Call("viewport", x1, y1, x2, y2)
}

func (gl *WebGL) CreateTexture() Texture {
	tex := gl.gl.Call("createTexture")
	return Texture(&tex)
}

func (gl *WebGL) BindTexture(texType TextureType, tex Texture) {
	if tex == nil {
		gl.gl.Call("bindTexture", int(texType), nil)
		return
	}
	gl.gl.Call("bindTexture", int(texType), js.Value(*tex))
}

func (gl *WebGL) TexImage2D(texType TextureType, level int, internalFmt, fmt PixelFormat, typ Type, img interface{}) {
	gl.gl.Call("texImage2D", int(texType), level, int(internalFmt), int(fmt), int(typ), img)
}

func (gl *WebGL) TexParameteri(texType TextureType, param TextureParameter, val interface{}) {
	gl.gl.Call("texParameteri", int(texType), int(param), val)
}

func (gl *WebGL) ActiveTexture(i TextureNumber) {
	gl.gl.Call("activeTexture", int(i))
}

func (gl *WebGL) BlendFunc(s, d BlendFactor) {
	gl.gl.Call("blendFunc", int(s), int(d))
}
