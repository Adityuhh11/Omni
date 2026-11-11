import { test } from "node:test";
import assert from "node:assert/strict";
import { colorsFor, folderName } from "./folders.ts";

test("each folder keeps its own colour, a ninth and a gone one are grey, a hidden one is invisible", () => {
  const roots = ["a", "b", "c", "d", "e", "f", "g", "h", "i", ""];
  const source = new Uint16Array([0, 1, 8, 9, 1]);
  const colors = colorsFor(source, roots, new Set([1]));
  const at = (i: number) => Array.from(colors.slice(i * 4, i * 4 + 4));
  assert.notDeepEqual(at(0).slice(0, 3), at(1).slice(0, 3));
  assert.deepEqual(at(2).slice(0, 3), at(3).slice(0, 3), "the ninth folder and a gone one share the grey");
  assert.equal(at(1)[3], 0);
  assert.equal(at(4)[3], 0);
  assert.ok(at(0)[3] > 0);
});

test("a folder is named by the last part of its path, on either system", () => {
  assert.equal(folderName(String.raw`C:\Users\me\Documents\ `.trim()), "Documents");
  assert.equal(folderName("/Users/me/Notes"), "Notes");
  assert.equal(folderName(""), "No longer searched");
});
