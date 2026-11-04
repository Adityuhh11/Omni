// The days the clocks change, in a zone that has them, whatever zone the machine running the tests is in.
process.env.TZ = "America/New_York";

import { test } from "node:test";
import assert from "node:assert/strict";
import { placeDay } from "./calendar.ts";

const ev = (id: string, start: Date, end: Date) => ({ id, start, end, allDay: false });

test("on the day the clocks go back, events sit at the hour the clock shows", () => {
  const day = new Date(2026, 10, 1); // Sunday 1 November 2026: 25 hours long
  const placed = placeDay([ev("tea", new Date(2026, 10, 1, 15), new Date(2026, 10, 1, 16)), ev("late", new Date(2026, 10, 1, 23), new Date(2026, 10, 1, 23, 45))], day);
  assert.deepEqual(placed.map((p) => [p.event.id, p.from, p.to]), [["tea", 900, 960], ["late", 1380, 1425]]);
});

test("on the day the clocks go forward, events sit at the hour the clock shows", () => {
  const day = new Date(2026, 2, 8); // Sunday 8 March 2026: 23 hours long
  const placed = placeDay([ev("lunch", new Date(2026, 2, 8, 12), new Date(2026, 2, 8, 13))], day);
  assert.deepEqual(placed.map((p) => [p.from, p.to]), [[720, 780]]);
});
