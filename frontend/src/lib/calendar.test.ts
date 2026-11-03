import { test } from "node:test";
import assert from "node:assert/strict";
import { colorOf, covers, placeDay, weekDays, weekStart, type Span } from "./calendar.ts";

const at = (d: number, h: number, m = 0) => new Date(2026, 10, d, h, m); // November 2026; the 2nd is a Monday
const ev = (id: string, start: Date, end: Date, allDay = false): Span => ({ id, start, end, allDay });

test("a week starts on its Monday, whatever day is asked", () => {
  for (const day of [at(2, 9), at(4, 23, 59), at(8, 0)]) assert.deepEqual(weekStart(day), at(2, 0));
  assert.deepEqual(weekStart(at(9, 0)), at(9, 0));
  const days = weekDays(at(30, 0)); // the week runs into December
  assert.equal(days.length, 7);
  assert.deepEqual(days[6], new Date(2026, 11, 6));
});

test("events that overlap share the width, the rest have it all", () => {
  const placed = placeDay(
    [
      ev("a", at(3, 9), at(3, 10)),
      ev("b", at(3, 9, 30), at(3, 11)),
      ev("c", at(3, 10), at(3, 10, 30)), // fits in a's lane once a is over: still in the cluster of three
      ev("d", at(3, 13), at(3, 14)),
      ev("all", at(3, 0), at(4, 0), true),
    ],
    at(3, 0),
  );
  const by = Object.fromEntries(placed.map((p) => [p.event.id, `${p.lane}/${p.lanes} ${p.from}-${p.to}`]));
  assert.deepEqual(by, { a: "0/2 540-600", b: "1/2 570-660", c: "0/2 600-630", d: "0/1 780-840" });
});

test("a night event shows on both days, cut at midnight, and a short one keeps a usable height", () => {
  const night = ev("n", at(3, 22), at(4, 2));
  assert.deepEqual(placeDay([night], at(3, 0)).map((p) => [p.from, p.to]), [[1320, 1440]]);
  assert.deepEqual(placeDay([night], at(4, 0)).map((p) => [p.from, p.to]), [[0, 120]]);
  assert.deepEqual(placeDay([ev("s", at(3, 9), at(3, 9, 5))], at(3, 0)).map((p) => p.to - p.from), [20]);
  assert.equal(placeDay([night], at(5, 0)).length, 0);
});

test("all-day events cover each day they span", () => {
  const trip = ev("t", at(4, 0), at(7, 0), true);
  assert.deepEqual(weekDays(at(2, 0)).map((d) => covers(trip, d)), [false, false, true, true, true, false, false]);
});

test("calendar colours cycle", () => {
  assert.equal(colorOf(0), colorOf(6));
  assert.notEqual(colorOf(0), colorOf(1));
});
