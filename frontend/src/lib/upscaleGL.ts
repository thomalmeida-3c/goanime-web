// Lightweight real-time video "upscaler": GL_LINEAR does the actual size
// increase (rendering to a canvas bigger than the source lets the GPU's own
// bilinear sampling smooth it out), and this shader adds an edge-aware
// unsharp mask on top to recover the crispness plain upscaling loses — the
// same visual goal as Anime4K's refine pass, as a single cheap draw call
// instead of Anime4K's multi-pass pipeline. Real-time in the browser, but a
// simplified approximation, not a port of the original algorithm.

const VERTEX_SRC = `#version 300 es
in vec2 a_position;
out vec2 v_uv;
void main() {
  v_uv = a_position * 0.5 + 0.5;
  gl_Position = vec4(a_position, 0.0, 1.0);
}`;

const FRAGMENT_SRC = `#version 300 es
precision highp float;
in vec2 v_uv;
uniform sampler2D u_tex;
uniform vec2 u_texel;
uniform float u_strength;
out vec4 outColor;

void main() {
  vec3 center = texture(u_tex, v_uv).rgb;
  vec3 n = texture(u_tex, v_uv + vec2(0.0, -u_texel.y)).rgb;
  vec3 s = texture(u_tex, v_uv + vec2(0.0,  u_texel.y)).rgb;
  vec3 e = texture(u_tex, v_uv + vec2( u_texel.x, 0.0)).rgb;
  vec3 w = texture(u_tex, v_uv + vec2(-u_texel.x, 0.0)).rgb;

  vec3 blur = (n + s + e + w + center) / 5.0;
  vec3 sharpened = center + (center - blur) * u_strength;

  // Only sharpen where there's real local contrast (an edge), so flat color
  // fields and compression noise don't get amplified along with line art.
  float edge = length(center - blur);
  float mask = smoothstep(0.01, 0.06, edge);

  outColor = vec4(clamp(mix(center, sharpened, mask), 0.0, 1.0), 1.0);
}`;

export function supportsWebGL2(): boolean {
  if (typeof window === "undefined") return false;
  try {
    const canvas = document.createElement("canvas");
    return !!(window.WebGL2RenderingContext && canvas.getContext("webgl2"));
  } catch {
    return false;
  }
}

function compileShader(gl: WebGL2RenderingContext, type: number, source: string): WebGLShader {
  const shader = gl.createShader(type);
  if (!shader) throw new Error("failed to create shader");
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const log = gl.getShaderInfoLog(shader);
    gl.deleteShader(shader);
    throw new Error(`shader compile error: ${log}`);
  }
  return shader;
}

export type UpscalePipeline = {
  render: (video: HTMLVideoElement, strength: number) => void;
  setSize: (width: number, height: number) => void;
  dispose: () => void;
};

// createUpscalePipeline sets up the GL program/buffers/texture once and
// returns a render() you can call every frame from a rAF/rVFC loop.
export function createUpscalePipeline(canvas: HTMLCanvasElement): UpscalePipeline | null {
  const gl = canvas.getContext("webgl2", { antialias: false, alpha: false });
  if (!gl) return null;

  const vertexShader = compileShader(gl, gl.VERTEX_SHADER, VERTEX_SRC);
  const fragmentShader = compileShader(gl, gl.FRAGMENT_SHADER, FRAGMENT_SRC);
  const program = gl.createProgram();
  if (!program) return null;
  gl.attachShader(program, vertexShader);
  gl.attachShader(program, fragmentShader);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    throw new Error(`program link error: ${gl.getProgramInfoLog(program)}`);
  }

  const quad = new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]);
  const vao = gl.createVertexArray();
  gl.bindVertexArray(vao);
  const buffer = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, quad, gl.STATIC_DRAW);
  const positionLoc = gl.getAttribLocation(program, "a_position");
  gl.enableVertexAttribArray(positionLoc);
  gl.vertexAttribPointer(positionLoc, 2, gl.FLOAT, false, 0, 0);

  const texture = gl.createTexture();
  gl.bindTexture(gl.TEXTURE_2D, texture);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);

  const texelLoc = gl.getUniformLocation(program, "u_texel");
  const strengthLoc = gl.getUniformLocation(program, "u_strength");

  gl.useProgram(program);
  gl.bindVertexArray(vao);
  gl.bindTexture(gl.TEXTURE_2D, texture);

  return {
    setSize(width, height) {
      canvas.width = width;
      canvas.height = height;
    },
    render(video, strength) {
      if (video.readyState < video.HAVE_CURRENT_DATA) return;
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGB, gl.RGB, gl.UNSIGNED_BYTE, video);
      gl.uniform2f(texelLoc, 1 / video.videoWidth, 1 / video.videoHeight);
      gl.uniform1f(strengthLoc, strength);
      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    },
    dispose() {
      gl.deleteTexture(texture);
      gl.deleteBuffer(buffer);
      gl.deleteVertexArray(vao);
      gl.deleteProgram(program);
      gl.deleteShader(vertexShader);
      gl.deleteShader(fragmentShader);
    },
  };
}

// scaleFactorFor picks how much bigger than the source the render target
// should be. Already-HD sources get little/no benefit (and cost a lot more
// GPU time), so the factor tapers off as the source gets bigger.
export function scaleFactorFor(sourceWidth: number): number {
  if (sourceWidth < 960) return 2.0;
  if (sourceWidth < 1600) return 1.5;
  return 1.0;
}
