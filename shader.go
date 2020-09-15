package main

const vsSource = `#version 300 es
	layout (location = 0) in vec4 aVertexPosition;
	layout (location = 1) in uint aVertexLabel;
	uniform mat4 uModelViewMatrix;
	uniform mat4 uProjectionMatrix;
	uniform mat4 uSelectMatrix;
	uniform vec3 uSelectRange;
	uniform float uZMin;
	uniform float uZRange;
	out lowp vec4 vColor;
	vec4 viewPosition;
	vec4 selectPosition;
	lowp float c;
	lowp float cSelected;

	void main(void) {
		viewPosition = uModelViewMatrix * aVertexPosition;
		gl_Position = uProjectionMatrix * viewPosition;
		gl_PointSize = clamp(20.0 / length(viewPosition), 1.0, 5.0);

		selectPosition = uSelectMatrix * aVertexPosition;
		if (uSelectMatrix[3][3] == 1.0 &&
				0.0 < selectPosition[0] && selectPosition[0] < uSelectRange[0] &&
				0.0 < selectPosition[1] && selectPosition[1] < uSelectRange[1] &&
				-uSelectRange[2] < selectPosition[2] && selectPosition[2] < uSelectRange[2]) {
			cSelected = 0.5;
		} else {
			cSelected = 0.0;
		}
		c = (aVertexPosition[2] - uZMin) / uZRange;
		if (aVertexLabel == 0u) {
			vColor = vec4(c, cSelected, 1.0 - c, 1.0);
		} else {
			vColor = vec4(1.0 - c, c, cSelected, 1.0);
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
		gl_PointSize = 5.5;

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
