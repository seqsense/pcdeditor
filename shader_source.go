package main

const vsSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	layout (location = 1) in uint aVertexLabel;
	layout (location = 2) in uint aSelectMask;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	uniform mat4 uSelectMatrix;
	uniform mat4 uCropMatrix;
	uniform float uZMin;
	uniform float uZRange;
	uniform float uPointSizeBase;
	uniform int uUseSelectMask;
	vec4 viewPosition;
	vec4 selectPosition;
	vec4 cropPosition;
	lowp float c;
	lowp float cSelected;
	out lowp vec4 vColor;

	void main(void) {
		cropPosition = uCropMatrix * aVertexPosition;
		if (any(lessThan(vec3(cropPosition), vec3(0, 0, 0))) ||
				any(lessThan(vec3(1.0, 1.0, 1.0), vec3(cropPosition)))) {
			gl_Position = vec4(-1, -1, 0, 0);
			gl_PointSize = 0.0;
			return;
		}

		viewPosition = uModelViewMatrix * aVertexPosition;
		gl_Position = uProjectionMatrix * viewPosition;

		if (uProjectionMatrix[3][3] == 0.0) {
			// Perspective mode
			gl_PointSize = clamp(uPointSizeBase / length(viewPosition), 1.0, uPointSizeBase);
		} else {
			// Orthographic mode
			gl_PointSize = uPointSizeBase / 20.0;
		}

		if (uUseSelectMask == 0) {
			selectPosition = uSelectMatrix * aVertexPosition;
			if (uSelectMatrix[3][3] == 1.0 &&
					all(lessThanEqual(vec3(0, 0, 0), vec3(selectPosition))) &&
					all(lessThanEqual(vec3(selectPosition), vec3(1.0, 1.0, 1.0)))) {
				cSelected = 0.5;
			} else {
				cSelected = 0.0;
			}
		} else {
			if ((aSelectMask & 0x10u) != 0u) {
				cSelected = 0.5;
			} else {
				cSelected = 0.0;
			}
		}
		c = (aVertexPosition[2] - uZMin) / uZRange;
		if (aVertexLabel == 0u) {
			vColor = vec4(c, cSelected, 1.0 - c, 1.0);
		} else {
			vColor = vec4(1.0 - c, c, cSelected, 1.0);
		}
	}
`

const vsSubSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	uniform float uPointSizeBase;
	vec4 viewPosition;
	out lowp vec4 vColor;

	void main(void) {
		viewPosition = uModelViewMatrix * aVertexPosition;
		gl_Position = uProjectionMatrix * viewPosition;

		if (uProjectionMatrix[3][3] == 0.0) {
			// Perspective mode
			gl_PointSize = clamp(uPointSizeBase / length(viewPosition), 1.0, uPointSizeBase);
		} else {
			// Orthographic mode
			gl_PointSize = uPointSizeBase / 20.0;
		}

		vColor = vec4(0.8, 0.8, 0.8, 0.5);
	}
`

const vsSelectSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	uniform float uPointSizeBase;
	vec4 viewPosition;
	out lowp vec4 vColor;

	void main(void) {
		viewPosition = uModelViewMatrix * aVertexPosition;
		gl_Position = uProjectionMatrix * viewPosition;
		if (uProjectionMatrix[3][3] == 0.0) {
			// Perspective mode
			gl_PointSize = clamp(1.5 * uPointSizeBase / length(viewPosition), 6.0, uPointSizeBase);
		} else {
			// Orthographic mode
			gl_PointSize = 3.0 * uPointSizeBase / 20.0;
		}

		vColor = vec4(1.0, 1.0, 1.0, 0.8);
	}
`

const fsSource = `#version 300 es
	in lowp vec4 vColor;
	out lowp vec4 outColor;

	void main(void) {
		outColor = vColor;
	}
`

const vsMapSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	layout (location = 1) in vec2 aTextureCoord;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	out highp vec2 vTextureCoord;

	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		vTextureCoord = aTextureCoord;
	}
`

const fsMapSource = `#version 300 es
	in highp vec2 vTextureCoord;
	uniform sampler2D uSampler;
	uniform lowp float uAlpha;
	out lowp vec4 vColor;

	void main(void) {
		vColor = texture(uSampler, vec2(vTextureCoord.s, vTextureCoord.t));
		vColor[3] = uAlpha;
	}
`

const csComputeSelectSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	layout (location = 2) in uint aSelectMask;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	uniform mat4 uSelectMatrix;
	uniform mat4 uCropMatrix;
	uniform vec3 uOrigin;
	uniform vec3 uDir;
	lowp vec4 selectPosition;
	lowp vec4 cropPosition;
	lowp vec3 rel;
	lowp float dotDir;
	lowp float distSq;
	lowp float dSq;
	vec4 screenPosition;
	flat out uint oResult;

	void main(void) {
		oResult = aSelectMask & 0x10u; // keep previous segment select result

		cropPosition = uCropMatrix * aVertexPosition;
		if (any(lessThan(vec3(cropPosition), vec3(0, 0, 0))) ||
				any(lessThan(vec3(1.0, 1.0, 1.0), vec3(cropPosition)))) {
			oResult |= 1u;
		} else {
			screenPosition = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
			if (any(lessThan(vec3(screenPosition), vec3(0, 0, 0))) ||
					any(lessThan(vec3(1.0, 1.0, 1.0), vec3(screenPosition)))) {
				oResult |= 8u;
			}
		}

		selectPosition = uSelectMatrix * aVertexPosition;
		if (uSelectMatrix[3][3] == 1.0 &&
				all(lessThanEqual(vec3(0, 0, 0), vec3(selectPosition))) &&
				all(lessThanEqual(vec3(selectPosition), vec3(1.0, 1.0, 1.0)))) {
			oResult |= 2u;
		}

		// Check distance from mouse cursor
		rel = vec3(uOrigin) - vec3(aVertexPosition);
		dotDir = dot(rel, vec3(uDir));
		if (dotDir < 0.0) {
			distSq = dot(rel, rel);
			dSq = distSq - dotDir * dotDir;
			if (dSq < 0.1 * 0.1 && distSq > 1.0) {
				oResult |= 4u;
			}
		}
	}
`

const fsComputeSelectSource = `#version 300 es
	void main(void) {
	}
`
