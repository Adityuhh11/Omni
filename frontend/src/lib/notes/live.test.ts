import { test } from "node:test";
import assert from "node:assert/strict";
import { EditorState } from "@codemirror/state";
import { decorate, continueList, liveEditor } from "./live.ts";

const open = () => {};
const state = (doc: string, caret: number) => EditorState.create({ doc, selection: { anchor: caret }, extensions: liveEditor({ placeholder: "", openLink: open, label: "" }) });

/** What is hidden or drawn as a widget, as the text it stands for, and the classes laid over text. */
function shown(doc: string, caret: number) {
  const hidden: string[] = [];
  const classes: string[] = [];
  const set = decorate({ state: state(doc, caret), visibleRanges: [{ from: 0, to: doc.length }] });
  set.between(0, doc.length, (from, to, value) => {
    if (value.spec.class) classes.push(value.spec.class);
    else if (to > from) hidden.push(doc.slice(from, to));
  });
  return { hidden, classes };
}

test("the markup is hidden everywhere but under the caret", () => {
  const doc = "# Title\n\nsome **bold** and [a link](http://x.com)";
  const away = shown(doc, 0); // on the heading: its mark shows, the rest of the note is rendered
  assert.ok(!away.hidden.includes("# "));
  assert.ok(away.hidden.includes("**") && away.hidden.includes("[") && away.hidden.includes("](http://x.com)"));
  const below = shown(doc, doc.length); // in the link: the link's markup shows, the heading's is hidden again
  assert.ok(below.hidden.includes("# "));
  assert.ok(below.hidden.includes("**") && !below.hidden.includes("["));
  assert.ok(below.classes.includes("cm-h cm-h1") && below.classes.includes("cm-strong") && below.classes.includes("cm-link"));
});

test("a task is a box, a done one is struck through, a bullet is a dot", () => {
  const { hidden, classes } = shown("- [ ] open\n- [x] shut\n- plain", 0);
  assert.ok(hidden.includes("[x]") && classes.includes("cm-done"));
  assert.ok(hidden.includes("-"), "the plain item's bullet is replaced");
  assert.doesNotThrow(() => shown("- [x]", 0), "a done task with no words after its box");
});

/** Runs Enter's command on a note and gives the text and the caret after it. */
function enter(doc: string, caret = doc.length) {
  let current = state(doc, caret);
  const view = { get state() { return current; }, dispatch: (spec: Parameters<EditorState["update"]>[0]) => (current = current.update(spec).state) };
  const handled = continueList(view);
  return { handled, text: current.doc.toString(), caret: current.selection.main.head };
}

test("Enter goes on with a list, numbers it, keeps a task a task and ends it on an empty item", () => {
  assert.equal(enter("- one").text, "- one\n- ");
  assert.equal(enter("2. two").text, "2. two\n3. ");
  assert.equal(enter("  - [x] done").text, "  - [x] done\n  - [ ] ");
  assert.equal(enter("- one\n- ").text, "- one\n");
  assert.equal(enter("plain text").handled, false);
  assert.equal(enter("- one", 1).handled, false, "a caret before the marker's end just breaks the line");
});
