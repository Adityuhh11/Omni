/**
 * Draws the map's points with WebGL2: one soft round dot per chunk, added onto the dark (so dense regions glow), smaller with
 * distance. Points a query landed among are "lit": larger, with a white core, while the rest dim.
 */

const VERTEX = `#version 300 es
in vec3 a_pos;
in vec4 a_color;
in float a_lit;
uniform mat4 u_matrix;
uniform float u_flat;
uniform float u_size;
uniform float u_dim;
out vec4 v_color;
out float v_lit;
void main() {
  gl_Position = u_matrix * vec4(a_pos.xy, a_pos.z * (1.0 - u_flat), 1.0);
  gl_PointSize = clamp(u_size * (1.0 + a_lit * 1.8) / gl_Position.w, 1.5, 30.0);
  v_color = vec4(a_color.rgb, a_color.a * mix(u_dim, 1.0, a_lit));
  v_lit = a_lit;
}`;

const FRAGMENT = `#version 300 es
precision mediump float;
in vec4 v_color;
in float v_lit;
out vec4 color;
void main() {
  float d = length(gl_PointCoord - 0.5) * 2.0;
  if (d > 1.0) discard;
  float core = v_lit * (1.0 - smoothstep(0.0, 0.55, d));
  color = vec4(mix(v_color.rgb, vec3(1.0), core * 0.7), v_color.a * (1.0 - smoothstep(0.45, 1.0, d)));
}`;

export interface Renderer {
  setPoints(coords: Float32Array, colors: Float32Array): void;
  setColors(colors: Float32Array): void;
  setLit(lit: Float32Array): void;
  draw(matrix: Float32Array, flat: number, dim: number, size: number): void;
}

/** A renderer for canvas, or null when the machine has no WebGL2 (the page then says so). */
export function createRenderer(canvas: HTMLCanvasElement): Renderer | null {
  const gl = canvas.getContext("webgl2", { antialias: false, premultipliedAlpha: false, alpha: false });
  if (!gl) return null;
  const program = link(gl, VERTEX, FRAGMENT);
  const at = (name: string) => gl.getUniformLocation(program, name);
  const uniforms = { matrix: at("u_matrix"), flat: at("u_flat"), size: at("u_size"), dim: at("u_dim") };
  const vao = gl.createVertexArray();
  gl.bindVertexArray(vao);
  const buffer = (name: string, size: number) => {
    const b = gl.createBuffer();
    const location = gl.getAttribLocation(program, name);
    gl.bindBuffer(gl.ARRAY_BUFFER, b);
    gl.enableVertexAttribArray(location);
    gl.vertexAttribPointer(location, size, gl.FLOAT, false, 0, 0);
    return b;
  };
  const positions = buffer("a_pos", 3);
  const colours = buffer("a_color", 4);
  const lights = buffer("a_lit", 1);
  let count = 0;

  const upload = (b: WebGLBuffer | null, data: Float32Array) => {
    gl.bindBuffer(gl.ARRAY_BUFFER, b);
    gl.bufferData(gl.ARRAY_BUFFER, data, gl.STATIC_DRAW);
  };

  return {
    setPoints(coords, colors) {
      count = coords.length / 3;
      upload(positions, coords);
      upload(colours, colors);
      upload(lights, new Float32Array(count));
    },
    setColors: (colors) => upload(colours, colors),
    setLit: (lit) => upload(lights, lit),
    draw(matrix, flat, dim, size) {
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.clearColor(0.055, 0.059, 0.071, 1); // the map's night (#0e0f12), whatever the system's look
      gl.clear(gl.COLOR_BUFFER_BIT);
      gl.enable(gl.BLEND);
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
      gl.useProgram(program);
      gl.uniformMatrix4fv(uniforms.matrix, false, matrix);
      gl.uniform1f(uniforms.flat, flat);
      gl.uniform1f(uniforms.size, size);
      gl.uniform1f(uniforms.dim, dim);
      gl.bindVertexArray(vao);
      gl.drawArrays(gl.POINTS, 0, count);
    },
  };
}

function link(gl: WebGL2RenderingContext, vertex: string, fragment: string): WebGLProgram {
  const program = gl.createProgram();
  for (const [type, source] of [
    [gl.VERTEX_SHADER, vertex],
    [gl.FRAGMENT_SHADER, fragment],
  ] as const) {
    const shader = gl.createShader(type)!;
    gl.shaderSource(shader, source);
    gl.compileShader(shader);
    if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) throw new Error(`map shader: ${gl.getShaderInfoLog(shader)}`);
    gl.attachShader(program, shader);
  }
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) throw new Error(`map program: ${gl.getProgramInfoLog(program)}`);
  return program;
}
