import assert from "node:assert/strict";
import { test } from "node:test";
import { after, before, stepsFor } from "./flow.ts";

test("file search adds the folders screen", () => {
  assert.deepEqual(stepsFor(true), ["welcome", "hotkey", "look", "features", "files", "done"]);
  assert.deepEqual(stepsFor(false), ["welcome", "hotkey", "look", "features", "done"]);
});

test("walking forward and back stays inside the screens", () => {
  assert.equal(after("features", true), "files");
  assert.equal(after("features", false), "done");
  assert.equal(after("done", true), "done");
  assert.equal(before("done", false), "features");
  assert.equal(before("done", true), "files");
  assert.equal(before("welcome", true), "welcome");
});
