import assert from "node:assert/strict";
import { test } from "node:test";
import { addressHost, escapeStep, kilo, lastQuestion, meter, sends, thoughtFor, toolGlyph, toolHint, unloadsIn, waiting } from "./chat.ts";

test("tokens are written short", () => {
  assert.equal(kilo(0), "0");
  assert.equal(kilo(640), "640");
  assert.equal(kilo(999), "999");
  assert.equal(kilo(1000), "1k");
  assert.equal(kilo(3100), "3.1k");
  assert.equal(kilo(8192), "8.2k");
  assert.equal(meter(3100, 8192), "3.1k / 8.2k");
  assert.equal(meter(0, 4096), "0 / 4.1k");
});

test("thinking time is in seconds, then minutes", () => {
  assert.equal(thoughtFor(100), "Thought for 1 s");
  assert.equal(thoughtFor(4200), "Thought for 4 s");
  assert.equal(thoughtFor(61_000), "Thought for 1 min 1 s");
});

test("the unload time counts up to the minute and says nothing when the model stays", () => {
  const now = 1_000_000;
  assert.equal(unloadsIn(0, now), "");
  assert.equal(unloadsIn(now + 7 * 60_000 - 1, now), "unloads in 7 min");
  assert.equal(unloadsIn(now + 5_000, now), "unloads in 1 min");
  assert.equal(unloadsIn(now - 5_000, now), "unloads in 1 min");
});

test("Up recalls the last question, not the last answer", () => {
  assert.equal(lastQuestion(null), "");
  assert.equal(lastQuestion([]), "");
  assert.equal(
    lastQuestion([
      { role: "user", text: "first" },
      { role: "assistant", text: "a" },
      { role: "user", text: "second" },
      { role: "assistant", text: "b" },
    ]),
    "second",
  );
});

test("Escape stops an answer, then closes the history, then leaves", () => {
  assert.equal(escapeStep(true, true, "x"), "stop");
  assert.equal(escapeStep(false, true, ""), "history");
  assert.equal(escapeStep(false, false, "  "), "close");
  assert.equal(escapeStep(false, false, "half a question"), "none"); // a typed question is not thrown away by a stray Escape
});

test("Enter sends unless Shift is held, an input method is composing, or the box is empty", () => {
  const key = (extra: object = {}) => ({ key: "Enter", shiftKey: false, isComposing: false, ...extra });
  assert.equal(sends(key(), "hello"), true);
  assert.equal(sends(key({ shiftKey: true }), "hello"), false);
  assert.equal(sends(key({ isComposing: true }), "hello"), false);
  assert.equal(sends(key(), "   "), false);
  assert.equal(sends(key({ key: "a" }), "hello"), false);
});

test("the waiting line prefers what the program said", () => {
  assert.equal(waiting("loading", "The graphics card ran out of memory; starting on the processor…"), "The graphics card ran out of memory; starting on the processor…");
  assert.equal(waiting("loading", ""), "Starting the model…");
  assert.equal(waiting("thinking", ""), "Thinking…");
  assert.equal(waiting("reading", ""), "Reading…");
  assert.equal(waiting("", ""), "Reading…");
});

test("a tool group's glyph and hint say what it is and whether it asks", () => {
  assert.equal(toolGlyph("files"), "file");
  assert.equal(toolGlyph("browser"), "web");
  assert.match(toolHint({ group: "browser", writes: false, always: false }), /visited or bookmarked/);
  assert.equal(toolGlyph("script:backup"), "code");
  assert.equal(toolGlyph("calendar"), "calendar");
  assert.equal(toolGlyph("something-new"), "spark");
  assert.match(toolHint({ group: "script:backup", writes: true, always: false }), /Asks before each run/);
  assert.match(toolHint({ group: "script:backup", writes: true, always: true }), /without asking/);
  assert.match(toolHint({ group: "notes", writes: true, always: false }), /Asks before adding/);
});

test("an address is local only when it names this computer", () => {
  assert.deepEqual(addressHost("http://localhost:1234/v1"), { host: "localhost:1234", local: true });
  assert.equal(addressHost("http://127.0.0.1:11434").local, true);
  assert.equal(addressHost("http://[::1]:8080").local, true);
  assert.deepEqual(addressHost(" https://api.openai.com/v1 "), { host: "api.openai.com", local: false });
  assert.equal(addressHost("http://192.168.1.5:1234").local, false);
  assert.equal(addressHost("http://localhost.evil.test").local, false);
  assert.deepEqual(addressHost("not an address"), { host: "", local: false });
});
