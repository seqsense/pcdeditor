package main

const vsSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	layout (location = 1) in uint aVertexLabel;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	const float zMax = 5.0;
	const float zMin = -5.0;
	const float zRange = zMax - zMin;
	lowp float c;
	out lowp vec4 vColor;

	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 2.0;

		c = (aVertexPosition[2] - zMin) / zRange;
		if (aVertexLabel == 0u) {
			vColor = vec4(c, 0.0, 1.0 - c, 1.0);
		} else {
			vColor = vec4(1.0 - c, c, 0.0, 1.0);
		}
	}
`

const vsSelectSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	out lowp vec4 vColor;

	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 3.0;

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
