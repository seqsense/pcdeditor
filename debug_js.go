package main

import (
	webgl "github.com/seqsense/webgl-go"
)

func showDebugInfo(gl *webgl.WebGL) {
	defer func() {
		if r := recover(); r != nil {
			println("Failed to get debug info")
		}
	}()

	ri, ok := gl.GetExtension("WEBGL_debug_renderer_info")
	if !ok {
		println("GPU info: hidden by the browser privacy setting")
		return
	}
	println("GPU:",
		gl.GetParameter(ri.Get("UNMASKED_VENDOR_WEBGL").Int()).String(),
		gl.GetParameter(ri.Get("UNMASKED_RENDERER_WEBGL").Int()).String(),
	)
	println("Max texture size:",
		gl.GetParameter(gl.JS().Get("MAX_TEXTURE_SIZE").Int()).Int(),
	)
}
