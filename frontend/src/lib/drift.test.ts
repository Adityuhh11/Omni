import { test } from "node:test";
import assert from "node:assert/strict";
import { nextDrift, SPEED, type Drift } from "./drift.ts";

test("a drifting light stays inside the box, turns gently and always moves at the same speed", () => {
  for (const [width, height] of [
    [850, 60],
    [850, 480],
    [300, 300],
  ]) {
    let at: Drift = { x: width / 2, y: height / 2, heading: 0 };
    for (let i = 0; i < 2000; i++) {
      const next = nextDrift(at, width, height);
      assert.ok(next.x >= 0 && next.x <= width && next.y >= 0 && next.y <= height, `${width}x${height}: ${next.x},${next.y}`);
      assert.ok(Math.abs(Math.hypot(next.x - at.x, next.y - at.y) / next.seconds - SPEED) < 1e-9);
      at = next;
    }
  }
});
