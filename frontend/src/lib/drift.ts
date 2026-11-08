/** Where a drifting light is: a point in CSS pixels and the direction it travels, in radians. */
export type Drift = { x: number; y: number; heading: number };

/** How fast the lights travel, in CSS pixels a second: slow enough to be calm. */
export const SPEED = 12;

/**
 * The next stretch of a light's path inside a box of width × height: a gentle turn (at most 35° either way) and 100 to 220 px on,
 * so the path bends like a current instead of zigzagging; at an edge it turns back inwards. Also says how long the stretch takes
 * at SPEED, so every light moves at the same constant pace. random is Math.random, or a fixed sequence in a test.
 */
export function nextDrift(from: Drift, width: number, height: number, random = Math.random): Drift & { seconds: number } {
  let heading = from.heading + (random() - 0.5) * ((70 * Math.PI) / 180);
  const length = 100 + random() * 120;
  let x = from.x + Math.cos(heading) * length;
  let y = from.y + Math.sin(heading) * length;
  if (x < 0 || x > width || y < 0 || y > height) {
    heading = Math.atan2(height / 2 - from.y, width / 2 - from.x) + (random() - 0.5) * 0.8; // back towards the middle
    x = Math.min(width, Math.max(0, from.x + Math.cos(heading) * length));
    y = Math.min(height, Math.max(0, from.y + Math.sin(heading) * length));
  }
  return { x, y, heading, seconds: Math.hypot(x - from.x, y - from.y) / SPEED };
}
