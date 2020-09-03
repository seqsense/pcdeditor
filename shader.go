package main

const vsSource = `
	attribute vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	varying lowp vec4 vColor;
	const float zMax = 5.0;
	const float zMin = -5.0;
	const float zRange = zMax - zMin;
	varying lowp float c;
	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 2.0;

		c = (aVertexPosition[2] - zMin) / zRange;
		vColor = vec4(c, 0.0, 1.0 - c, 1.0);
	}
`

const vsSelectSource = `
	attribute vec4 aVertexPosition;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	varying lowp vec4 vColor;
	void main(void) {
		gl_Position = uProjectionMatrix * uModelViewMatrix * aVertexPosition;
		gl_PointSize = 3.0;

		vColor = vec4(1.0, 1.0, 1.0, 0.8);
	}
`

const fsSource = `
	varying lowp vec4 vColor;
	void main(void) {
		gl_FragColor = vColor;
	}
`
