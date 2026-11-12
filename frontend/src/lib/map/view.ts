/**
 * The map's camera and the arithmetic around it: decoding the engine's points, the matrix that puts them on the screen,
 * and finding the point under the pointer. No WebGL here, so it is tested on its own.
 *
 * The world: points inside the unit ball (the engine centres and scales them). The camera orbits a target point. The flat
 * (2D) view is the same points with their third coordinate squeezed to nothing, seen straight down that axis; `flat` runs
 * from 0 (3D) to 1 (2D), so the change between them can be animated.
 */

export type Mat4 = Float32Array;

export interface Camera {
  yaw: number; // radians around the up axis
  pitch: number; // radians above the plane
  distance: number; // from the target
  target: [number, number, number];
  flat: number; // 0 = 3D, 1 = 2D
}

export const FOV = (50 * Math.PI) / 180;
export const MIN_DISTANCE = 0.08;
export const MAX_DISTANCE = 8;
export const MAX_PITCH = Math.PI / 2 - 0.01;

/** The engine's base64 bytes as numbers (little-endian, which every machine Omni runs on is). */
export function decodeFloats(base64: string): Float32Array {
  return new Float32Array(bytesOf(base64).buffer);
}

export function decodeUint16(base64: string): Uint16Array {
  return new Uint16Array(bytesOf(base64).buffer);
}

export function decodeUint32(base64: string): Uint32Array {
  return new Uint32Array(bytesOf(base64).buffer);
}

function bytesOf(base64: string): Uint8Array {
  const text = atob(base64);
  const bytes = new Uint8Array(text.length);
  for (let i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i);
  return bytes;
}

/** Where the camera is, in world space. */
export function eyeOf(c: Camera): [number, number, number] {
  const ground = Math.cos(c.pitch) * c.distance;
  return [
    c.target[0] + ground * Math.sin(c.yaw),
    c.target[1] + Math.sin(c.pitch) * c.distance,
    c.target[2] + ground * Math.cos(c.yaw),
  ];
}

/**
 * The matrix from a point (with its third coordinate already squeezed by `flat`) to clip space. Column-major, as WebGL
 * wants it. In the flat view the camera looks straight down the third axis, so "up" is the second.
 */
export function viewProjection(c: Camera, aspect: number): Mat4 {
  const eye = eyeOf(c);
  const target: [number, number, number] = [c.target[0], c.target[1], c.target[2] * (1 - c.flat)];
  const near = Math.max(0.001, c.distance * 0.01);
  return multiply(perspective(FOV, aspect, near, c.distance + 4), lookAt(eye, target, [0, 1, 0])); // pitch stays short of the pole
}

/** Where a point lands on a canvas of the given size, and how far it is (clip w): null when it is behind the camera. */
export function toScreen(m: Mat4, x: number, y: number, z: number, width: number, height: number): [number, number, number] | null {
  const cx = m[0] * x + m[4] * y + m[8] * z + m[12];
  const cy = m[1] * x + m[5] * y + m[9] * z + m[13];
  const cw = m[3] * x + m[7] * y + m[11] * z + m[15];
  if (cw <= 0) return null;
  return [((cx / cw + 1) / 2) * width, ((1 - cy / cw) / 2) * height, cw];
}

/**
 * The point nearest to (px, py) on the screen within `radius` pixels, or -1. Of two about as near, the one nearer the
 * camera wins: it is drawn on top. ponytail: every point is projected (about 1 ms for 50k); a grid would beat it past 500k.
 */
export function pick(
  m: Mat4,
  coords: Float32Array,
  flat: number,
  visible: (index: number) => boolean,
  px: number,
  py: number,
  width: number,
  height: number,
  radius = 10,
): number {
  let best = -1;
  let bestScore = Infinity;
  const squeeze = 1 - flat;
  for (let i = 0; i < coords.length / 3; i++) {
    if (!visible(i)) continue;
    const at = toScreen(m, coords[3 * i], coords[3 * i + 1], coords[3 * i + 2] * squeeze, width, height);
    if (!at) continue;
    const d = Math.hypot(at[0] - px, at[1] - py);
    if (d > radius) continue;
    const score = d + at[2] * 2; // a little depth decides between near neighbours
    if (score < bestScore) {
      bestScore = score;
      best = i;
    }
  }
  return best;
}

/** One step of easing `from` toward `to`: the same feel at any frame rate (`dt` in seconds, `rate` per second). */
export function approach(from: number, to: number, dt: number, rate = 9): number {
  const k = 1 - Math.exp(-rate * dt);
  const next = from + (to - from) * k;
  return Math.abs(to - next) < 1e-5 ? to : next;
}

export function clamp(value: number, low: number, high: number): number {
  return Math.min(high, Math.max(low, value));
}

function perspective(fov: number, aspect: number, near: number, far: number): Mat4 {
  const f = 1 / Math.tan(fov / 2);
  const out = new Float32Array(16);
  out[0] = f / aspect;
  out[5] = f;
  out[10] = (far + near) / (near - far);
  out[11] = -1;
  out[14] = (2 * far * near) / (near - far);
  return out;
}

function lookAt(eye: number[], target: number[], up: number[]): Mat4 {
  const z = normalize([eye[0] - target[0], eye[1] - target[1], eye[2] - target[2]]);
  const x = normalize(cross(up, z));
  const y = cross(z, x);
  const out = new Float32Array(16);
  out.set([x[0], y[0], z[0], 0, x[1], y[1], z[1], 0, x[2], y[2], z[2], 0]);
  out[12] = -dot(x, eye);
  out[13] = -dot(y, eye);
  out[14] = -dot(z, eye);
  out[15] = 1;
  return out;
}

function multiply(a: Mat4, b: Mat4): Mat4 {
  const out = new Float32Array(16);
  for (let col = 0; col < 4; col++) {
    for (let row = 0; row < 4; row++) {
      let sum = 0;
      for (let k = 0; k < 4; k++) sum += a[k * 4 + row] * b[col * 4 + k];
      out[col * 4 + row] = sum;
    }
  }
  return out;
}

function cross(a: number[], b: number[]): number[] {
  return [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]];
}

function dot(a: number[], b: number[]): number {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
}

function normalize(v: number[]): number[] {
  const length = Math.hypot(v[0], v[1], v[2]) || 1;
  return [v[0] / length, v[1] / length, v[2] / length];
}
