import { test } from "node:test";
import assert from "node:assert/strict";
import { AGE_RAMP, ageColors, colorsFor, folderName, OTHER } from "./folders.ts";
import { piecesOf } from "./files.ts";

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

test("by type, the group named other is grey like a ninth", () => {
  const colors = colorsFor(new Uint16Array([0, 1]), ["pdf", "other"], new Set());
  assert.deepEqual(Array.from(colors.slice(4, 7)), Array.from(colorsFor(new Uint16Array([8]), [..."abcdefghi"], new Set()).slice(0, 3)));
});

test("by age, the oldest is the ramp's first colour and the newest its last, by rank, with unknown dates grey", () => {
  const { colors, cuts } = ageColors(new Uint32Array([100, 1_000_000, 200, 0]));
  const at = (i: number) => Array.from(colors.slice(i * 4, i * 4 + 3)).map((v) => Math.round(v * 255));
  const hex = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  assert.deepEqual(at(0), hex(AGE_RAMP[0]));
  assert.deepEqual(at(1), hex(AGE_RAMP[AGE_RAMP.length - 1]));
  assert.deepEqual(at(3), hex(OTHER));
  assert.deepEqual(cuts, [100, 200, 1_000_000]);
});

test("a file's pieces come in reading order, and only that file's", () => {
  const file = new Uint32Array([3, 1, 3, 3, 2]);
  const rank = new Uint32Array([2, 0, 0, 1, 0]);
  assert.deepEqual(Array.from(piecesOf(file, rank, 3)), [2, 3, 0]);
});
