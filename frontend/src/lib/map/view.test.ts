import { test } from "node:test";
import assert from "node:assert/strict";
import { approach, decodeFloats, decodeUint16, pick, toScreen, viewProjection, type Camera } from "./view.ts";

const front: Camera = { yaw: 0, pitch: 0, distance: 3, target: [0, 0, 0], flat: 0 };

test("the engine's base64 decodes to the numbers it encoded", () => {
  const floats = new Float32Array([0.5, -1.25, 3]);
  const words = new Uint16Array([0, 7, 65535]);
  const encode = (bytes: ArrayBufferLike) => btoa(String.fromCharCode(...new Uint8Array(bytes)));
  assert.deepEqual(Array.from(decodeFloats(encode(floats.buffer))), [0.5, -1.25, 3]);
  assert.deepEqual(Array.from(decodeUint16(encode(words.buffer))), [0, 7, 65535]);
});

test("the target is in the middle of the screen, right is right and up is up", () => {
  const m = viewProjection(front, 1);
  const centre = toScreen(m, 0, 0, 0, 200, 100)!;
  assert.ok(Math.abs(centre[0] - 100) < 1e-3 && Math.abs(centre[1] - 50) < 1e-3, `${centre}`);
  assert.ok(toScreen(m, 0.5, 0, 0, 200, 100)![0] > 100);
  assert.ok(toScreen(m, 0, 0.5, 0, 200, 100)![1] < 50);
  assert.equal(toScreen(m, 0, 0, 5, 200, 100), null, "behind the camera");
});

test("picking finds the point under the pointer, skips hidden ones and misses empty space", () => {
  const coords = new Float32Array([0, 0, 0, 0.5, 0, 0, -0.5, 0.3, 0]);
  const m = viewProjection(front, 1);
  const all = () => true;
  const [x, y] = toScreen(m, 0.5, 0, 0, 400, 400)!;
  assert.equal(pick(m, coords, 0, all, x + 3, y - 2, 400, 400), 1);
  assert.equal(pick(m, coords, 0, (i) => i !== 1, x, y, 400, 400), -1);
  assert.equal(pick(m, coords, 0, all, 5, 5, 400, 400), -1);
});

test("the flat view ignores the third coordinate", () => {
  const flat: Camera = { ...front, flat: 1 };
  const m = viewProjection(flat, 1);
  const near = new Float32Array([0.2, 0.1, 0.9]);
  const [x, y] = toScreen(m, 0.2, 0.1, 0, 300, 300)!;
  assert.equal(pick(m, near, 1, () => true, x, y, 300, 300), 0);
});

test("easing reaches its goal and does not overshoot, whatever the frame rate", () => {
  for (const dt of [1 / 30, 1 / 60, 1 / 144]) {
    let v = 0;
    for (let t = 0; t < 2; t += dt) {
      v = approach(v, 1, dt);
      assert.ok(v <= 1);
    }
    assert.equal(v, 1);
  }
});
