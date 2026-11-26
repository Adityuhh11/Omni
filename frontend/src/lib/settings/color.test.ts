import { test } from "node:test";
import assert from "node:assert/strict";
import { hexToHsv, hsvToHex, normalizeHex } from "./color.ts";

test("a typed colour is read in any of the usual ways, and nothing else", () => {
  assert.equal(normalizeHex("#F2601D"), "#f2601d");
  assert.equal(normalizeHex("  f2601d "), "#f2601d");
  assert.equal(normalizeHex("#f60"), "#ff6600");
  for (const bad of ["", "#", "red", "#12345", "#1234567", "#12345g"]) assert.equal(normalizeHex(bad), null, bad);
});

test("a colour goes to hue, saturation and value and comes back the same", () => {
  for (const hex of ["#000000", "#ffffff", "#f2601d", "#3e63dd", "#46a758", "#e5484d", "#808080", "#0b0c0e"]) {
    assert.equal(hsvToHex(hexToHsv(hex)), hex);
  }
  assert.deepEqual(hexToHsv("#ff0000"), { h: 0, s: 1, v: 1 });
  assert.equal(hexToHsv("#00ff00").h, 120);
  assert.equal(hexToHsv("#0000ff").h, 240);
  assert.equal(hexToHsv("#777777").s, 0, "a grey has no saturation, whatever its hue");
});
