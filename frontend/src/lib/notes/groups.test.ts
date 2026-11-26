import { test } from "node:test";
import assert from "node:assert/strict";
import { dayGroup, grouped } from "./groups.ts";

const now = new Date(2026, 10, 26, 15, 0); // 26 November 2026, a Thursday afternoon
const ago = (days: number, hour = 9) => new Date(2026, 10, 26 - days, hour).getTime();

test("a note is grouped by how many days ago it was changed", () => {
  assert.equal(dayGroup(ago(0, 1), now), "Today");
  assert.equal(dayGroup(ago(1, 23), now), "Yesterday");
  assert.equal(dayGroup(ago(2), now), "Earlier this week");
  assert.equal(dayGroup(ago(6), now), "Earlier this week");
  assert.equal(dayGroup(ago(7), now), "Earlier this month");
  assert.equal(dayGroup(ago(29), now), "Earlier this month");
  assert.equal(dayGroup(ago(30), now), "Older");
});

test("newest first notes come out in runs, each under its own heading", () => {
  const notes = [{ Modified: ago(0) }, { Modified: ago(0, 2) }, { Modified: ago(1) }, { Modified: ago(40) }, { Modified: ago(90) }];
  assert.deepEqual(
    grouped(notes, now).map((g) => [g.heading, g.notes.length]),
    [["Today", 2], ["Yesterday", 1], ["Older", 2]],
  );
  assert.deepEqual(grouped([], now), []);
});
