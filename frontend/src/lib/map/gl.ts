/**
 * Draws the map's points with WebGL2: one solid round dot per chunk with a thin dark rim, smaller with distance. Dots are
 * opaque and depth-tested, so a near dot covers a far one instead of their light adding up into a haze. Points a query
 * landed among, or the pieces of a clicked file, are "lit": larger, with a white core, drawn in front; the rest fade
 * toward the background. A file's pieces are joined by a thin line in reading order.
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
out float v_size;
const vec3 NIGHT = vec3(0.055, 0.059, 0.071);
void main() {
  gl_Position = u_matrix * vec4(a_pos.xy, a_pos.z * (1.0 - u_flat), 1.0);
  gl_Position.z -= a_lit * 0.01 * gl_Position.w; // lit dots in front of the dimmed ones
  gl_PointSize = clamp(u_size * (1.0 + a_lit * 1.8) / gl_Position.w, 2.0, 30.0);
  v_color = vec4(mix(NIGHT, a_color.rgb, mix(u_dim, 1.0, a_lit)), a_color.a);
  v_lit = a_lit;
  v_size = gl_PointSize;
}`;

const FRAGMENT = `#version 300 es
precision mediump float;
in vec4 v_color;
in float v_lit;
in float v_size;
out vec4 color;
void main() {
  if (v_color.a < 0.5) discard; // a hidden folder
  float d = length(gl_PointCoord - 0.5) * 2.0;
  float px = 2.0 / v_size; // one screen pixel, in d units
  float edge = 1.0 - smoothstep(1.0 - px, 1.0, d);
  if (edge <= 0.0) discard;
  float rim = v_size > 5.0 ? smoothstep(1.0 - 2.5 * px, 1.0 - 1.5 * px, d) : 0.0;
  float core = v_lit * (1.0 - smoothstep(0.0, 0.45, d));
  vec3 rgb = mix(v_color.rgb, vec3(1.0), core * 0.8);
  color = vec4(mix(rgb, rgb * 0.35, rim), edge);
}`;

// The path through a file's pieces: as far forward as the lit dots, so dimmed ones do not cut it.
const LINE_VERTEX = `#version 300 es
in vec3 a_pos;
uniform mat4 u_matrix;
uniform float u_flat;
void main() {
  gl_Position = u_matrix * vec4(a_pos.xy, a_pos.z * (1.0 - u_flat), 1.0);
  gl_Position.z -= 0.01 * gl_Position.w;
}`;

const LINE_FRAGMENT = `#version 300 es
precision mediump float;
out vec4 color;
void main() {
  color = vec4(1.0, 1.0, 1.0, 0.5);
}`;

export interface Renderer {
  setPoints(coords: Float32Array, colors: Float32Array): void;
  setColors(colors: Float32Array): void;
  setLit(lit: Float32Array): void;
  setPath(points: Uint32Array): void;
  draw(matrix: Float32Array, flat: number, dim: number, size: number): void;
}

/** A renderer for canvas, or null when the machine has no WebGL2 (the page then says so). */
export function createRenderer(canvas: HTMLCanvasElement): Renderer | null {
  const gl = canvas.getContext("webgl2", { antialias: false, premultipliedAlpha: false, alpha: false, depth: true });
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

  const line = link(gl, LINE_VERTEX, LINE_FRAGMENT);
  const lineUniforms = { matrix: gl.getUniformLocation(line, "u_matrix"), flat: gl.getUniformLocation(line, "u_flat") };
  const lineVao = gl.createVertexArray();
  gl.bindVertexArray(lineVao);
  gl.bindBuffer(gl.ARRAY_BUFFER, positions);
  const linePos = gl.getAttribLocation(line, "a_pos");
  gl.enableVertexAttribArray(linePos);
  gl.vertexAttribPointer(linePos, 3, gl.FLOAT, false, 0, 0);
  const path = gl.createBuffer();
  gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, path); // kept by lineVao
  gl.bindVertexArray(null);
  let pathLength = 0;

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
    setPath(points) {
      gl.bindVertexArray(lineVao);
      gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, points, gl.STATIC_DRAW);
      gl.bindVertexArray(null);
      pathLength = points.length;
    },
    draw(matrix, flat, dim, size) {
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.clearColor(0.055, 0.059, 0.071, 1); // the map's night (#0e0f12), whatever the system's look
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
      gl.enable(gl.DEPTH_TEST);
      gl.depthFunc(gl.LEQUAL);
      gl.enable(gl.BLEND); // only the one-pixel edge is see-through
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
      if (pathLength > 1) {
        gl.useProgram(line);
        gl.uniformMatrix4fv(lineUniforms.matrix, false, matrix);
        gl.uniform1f(lineUniforms.flat, flat);
        gl.bindVertexArray(lineVao);
        gl.drawElements(gl.LINE_STRIP, pathLength, gl.UNSIGNED_INT, 0);
      }
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
