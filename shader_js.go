package main

import (
	"errors"
	"syscall/js"

	webgl "github.com/seqsense/webgl-go"
)

var errContextLost = errors.New("WebGL context lost")

func initVertexShader(gl *webgl.WebGL, src string) (webgl.Shader, error) {
	s := gl.CreateShader(gl.VERTEX_SHADER)
	gl.ShaderSource(s, src)
	gl.CompileShader(s)
	if !gl.GetShaderParameter(s, gl.COMPILE_STATUS).(bool) {
		if gl.IsContextLost() {
			return webgl.Shader(js.Null()), errContextLost
		}
		return webgl.Shader(js.Null()), errors.New("compile failed (VERTEX_SHADER)")
	}
	return s, nil
}

func initFragmentShader(gl *webgl.WebGL, src string) (webgl.Shader, error) {
	s := gl.CreateShader(gl.FRAGMENT_SHADER)
	gl.ShaderSource(s, src)
	gl.CompileShader(s)
	if !gl.GetShaderParameter(s, gl.COMPILE_STATUS).(bool) {
		if gl.IsContextLost() {
			return webgl.Shader(js.Null()), errContextLost
		}
		return webgl.Shader(js.Null()), errors.New("compile failed (FRAGMENT_SHADER)")
	}
	return s, nil
}

func linkShaders(gl *webgl.WebGL, fbVarings []string, shaders ...webgl.Shader) (webgl.Program, error) {
	program := gl.CreateProgram()
	for _, s := range shaders {
		gl.AttachShader(program, s)
	}
	if len(fbVarings) > 0 {
		gl.TransformFeedbackVaryings(program, fbVarings, gl.SEPARATE_ATTRIBS)
	}
	gl.LinkProgram(program)
	if !gl.GetProgramParameter(program, gl.LINK_STATUS).(bool) {
		if gl.IsContextLost() {
			return webgl.Program(js.Null()), errContextLost
		}
		return webgl.Program(js.Null()), errors.New("link failed: " + gl.GetProgramInfoLog(program))
	}
	return program, nil
}
