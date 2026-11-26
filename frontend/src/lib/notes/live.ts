/**
 * A markdown editor that renders as you type, as Obsidian's live preview does: headings are big, bold is bold, links are links and a task
 * is a box, and the markup (`# `, `**`, `[…](…)`) shows only where the caret is, on its line or inside its span.
 *
 * The text in the editor is always the note's markdown, character for character: everything here is decoration over it.
 */
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { defineLanguageFacet, Language, languageDataProp, LanguageSupport, syntaxTree } from "@codemirror/language";
import { EditorState, Prec, type Range } from "@codemirror/state";
import { Decoration, type DecorationSet, EditorView, keymap, placeholder, ViewPlugin, type ViewUpdate, WidgetType } from "@codemirror/view";
import { GFM, parser } from "@lezer/markdown";

// The language is built from the markdown parser alone (not @codemirror/lang-markdown, which brings the HTML, CSS and JavaScript
// languages for the code blocks): a note needs its tree, not highlighted code.
const data = defineLanguageFacet();
const markdown = new LanguageSupport(new Language(data, parser.configure([GFM, { props: [languageDataProp.add({ Document: data })] }]), [], "markdown"));

/** A task's box: ticking it changes the `[ ]` in the text to `[x]` and back. */
class TaskBox extends WidgetType {
  readonly checked: boolean;
  constructor(checked: boolean) {
    super();
    this.checked = checked;
  }
  eq(other: TaskBox) {
    return other.checked === this.checked;
  }
  toDOM(view: EditorView) {
    const box = document.createElement("input");
    box.type = "checkbox";
    box.className = "cm-task";
    box.checked = this.checked;
    box.tabIndex = -1;
    box.addEventListener("mousedown", (event) => event.preventDefault()); // clicking it must not move the caret
    box.addEventListener("click", () => {
      const at = view.posAtDOM(box);
      view.dispatch({ changes: { from: at + 1, to: at + 2, insert: this.checked ? " " : "x" } });
    });
    return box;
  }
  ignoreEvent() {
    return true;
  }
}

/** A list's bullet, drawn as a dot where the `-` was. */
class Bullet extends WidgetType {
  toDOM() {
    const dot = document.createElement("span");
    dot.className = "cm-bullet";
    dot.textContent = "•";
    return dot;
  }
}

/** A horizontal rule: the `---` line drawn as a line. */
class Rule extends WidgetType {
  toDOM() {
    const rule = document.createElement("span");
    rule.className = "cm-rule";
    return rule;
  }
}

const hidden = Decoration.replace({});
const bullet = Decoration.replace({ widget: new Bullet() });
const rule = Decoration.replace({ widget: new Rule() });
const marked = (name: string, attributes?: Record<string, string>) => Decoration.mark({ class: name, attributes });
const lined = (name: string) => Decoration.line({ class: name });

/** The decorations for what is on screen, given where the caret and the selection are. */
export function decorate(view: Pick<EditorView, "state" | "visibleRanges">): DecorationSet {
  const { state } = view;
  const out: Range<Decoration>[] = [];
  const touches = (from: number, to: number) => state.selection.ranges.some((r) => r.from <= to && r.to >= from);
  const lineTouches = (pos: number) => {
    const line = state.doc.lineAt(pos);
    return touches(line.from, line.to);
  };
  const lines = (from: number, to: number, name: string) => {
    for (let n = state.doc.lineAt(from).number; n <= state.doc.lineAt(to).number; n++) out.push(lined(name).range(state.doc.line(n).from));
  };
  const hide = (from: number, to: number) => from < to && out.push(hidden.range(from, to));

  for (const { from, to } of view.visibleRanges) {
    syntaxTree(state).iterate({
      from,
      to,
      enter(node) {
        const { name } = node;
        const heading = /^(?:ATX|Setext)Heading(\d)$/.exec(name);
        if (heading) {
          out.push(lined(`cm-h cm-h${heading[1]}`).range(state.doc.lineAt(node.from).from));
        } else if (name === "HeaderMark" && /^ATXHeading/.test(node.node.parent?.name ?? "") && !lineTouches(node.from)) {
          hide(node.from, Math.min(node.to + 1, state.doc.lineAt(node.from).to)); // "# " with its space; a setext heading's underline stays
        } else if (name === "StrongEmphasis") out.push(marked("cm-strong").range(node.from, node.to));
        else if (name === "Emphasis") out.push(marked("cm-em").range(node.from, node.to));
        else if (name === "Strikethrough") out.push(marked("cm-strike").range(node.from, node.to));
        else if (name === "InlineCode") out.push(marked("cm-code").range(node.from, node.to));
        else if (name === "EmphasisMark" || name === "StrikethroughMark" || (name === "CodeMark" && node.node.parent?.name === "InlineCode")) {
          const owner = node.node.parent;
          if (owner && !touches(owner.from, owner.to)) hide(node.from, node.to);
        } else if (name === "FencedCode") {
          lines(node.from, node.to, "cm-fence");
        } else if (name === "CodeMark" && node.node.parent?.name === "FencedCode") {
          out.push(marked("cm-faint").range(node.from, node.to));
        } else if (name === "CodeInfo") out.push(marked("cm-faint").range(node.from, node.to));
        else if (name === "Link") {
          const parts: { from: number; to: number }[] = [];
          for (let child = node.node.firstChild; child; child = child.nextSibling) if (child.name === "LinkMark") parts.push(child);
          const url = node.node.getChild("URL");
          const label = parts.length >= 2 ? { from: parts[0].to, to: parts[1].from } : { from: node.from, to: node.to };
          if (label.from < label.to) out.push(marked("cm-link", url ? { "data-href": state.sliceDoc(url.from, url.to) } : undefined).range(label.from, label.to));
          if (parts.length >= 2 && !touches(node.from, node.to)) {
            hide(parts[0].from, parts[0].to); // "["
            hide(parts[1].from, node.to); // "](address "title")"
          }
        } else if (name === "URL" && node.node.parent?.name !== "Link") {
          out.push(marked("cm-link", { "data-href": state.sliceDoc(node.from, node.to) }).range(node.from, node.to));
        } else if (name === "Blockquote") lines(node.from, node.to, "cm-quote");
        else if (name === "QuoteMark" && !lineTouches(node.from)) hide(node.from, Math.min(node.to + 1, state.doc.lineAt(node.from).to));
        else if (name === "ListMark" && !lineTouches(node.from) && node.node.parent?.parent?.name === "BulletList") {
          const task = node.node.parent.getChild("Task");
          if (task) hide(node.from, Math.min(node.to + 1, state.doc.lineAt(node.from).to)); // the box stands in for the bullet
          else out.push(bullet.range(node.from, node.to));
        } else if (name === "TaskMarker") {
          const checked = /[xX]/.test(state.sliceDoc(node.from, node.to));
          if (!touches(node.from, node.to)) out.push(Decoration.replace({ widget: new TaskBox(checked) }).range(node.from, node.to));
          const task = node.node.parent;
          if (checked && task && task.to > node.to) out.push(marked("cm-done").range(node.to, task.to)); // a mark may not be empty
        } else if (name === "HorizontalRule" && !lineTouches(node.from)) out.push(rule.range(node.from, node.to));
        else if (name === "Table") lines(node.from, node.to, "cm-table");
      },
    });
  }
  return Decoration.set(out, true);
}

const preview = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    constructor(view: EditorView) {
      this.decorations = decorate(view);
    }
    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged || update.selectionSet || syntaxTree(update.startState) !== syntaxTree(update.state)) {
        this.decorations = decorate(update.view);
      }
    }
  },
  { decorations: (plugin) => plugin.decorations },
);

const LIST = /^(\s*)(?:([-*+])|(\d+)([.)]))( \[[ xX]\])?( +)/;

/** Enter at the end of a list item starts the next item; Enter on an empty one ends the list. Anywhere else it is a plain new line. */
export function continueList(view: Pick<EditorView, "state" | "dispatch">): boolean {
  const { state } = view;
  const range = state.selection.main;
  if (!range.empty) return false;
  const line = state.doc.lineAt(range.head);
  const found = LIST.exec(line.text);
  if (!found || range.head < line.from + found[0].length) return false;
  const [whole, indent, bullet, number, delimiter, task, gap] = found;
  if (line.text.length === whole.length) {
    // nothing after the marker: the list is over, and the marker goes
    view.dispatch({ changes: { from: line.from, to: line.to, insert: indent }, selection: { anchor: line.from + indent.length } });
    return true;
  }
  const next = `${indent}${bullet ?? `${Number(number) + 1}${delimiter}`}${task ? " [ ]" : ""}${gap}`;
  view.dispatch({ changes: { from: range.head, insert: `\n${next}` }, selection: { anchor: range.head + 1 + next.length }, scrollIntoView: true });
  return true;
}

const theme = EditorView.theme({
  "&": { height: "100%", color: "var(--text)", backgroundColor: "transparent", fontSize: "14.5px" },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": { fontFamily: "var(--sans)", lineHeight: "1.7", overflow: "auto" },
  ".cm-content": { padding: "18px 24px 48px", caretColor: "var(--chrome)" },
  ".cm-line": { padding: "0" },
  "&.cm-focused .cm-cursor": { borderLeftColor: "var(--chrome)", borderLeftWidth: "2px" },
  ".cm-placeholder": { color: "var(--text-3)" },
  ".cm-h": { fontWeight: "650", letterSpacing: "-0.015em", lineHeight: "1.3" },
  ".cm-h1": { fontSize: "1.85em", fontWeight: "680", paddingTop: "0.35em" },
  ".cm-h2": { fontSize: "1.5em", paddingTop: "0.3em" },
  ".cm-h3": { fontSize: "1.25em", paddingTop: "0.2em" },
  ".cm-h4, .cm-h5, .cm-h6": { fontSize: "1.08em" },
  ".cm-strong": { fontWeight: "680" },
  ".cm-em": { fontStyle: "italic" },
  ".cm-strike": { textDecoration: "line-through", color: "var(--text-3)" },
  ".cm-code": { padding: "1px 5px", borderRadius: "6px", backgroundColor: "var(--tile)", fontFamily: "var(--mono)", fontSize: "0.88em" },
  ".cm-link": { color: "var(--chrome)", textDecoration: "underline", textUnderlineOffset: "2px" },
  ".cm-faint": { color: "var(--text-3)" },
  ".cm-done": { color: "var(--text-3)", textDecoration: "line-through" },
  ".cm-quote": { paddingLeft: "14px", borderLeft: "2px solid var(--text-3)", color: "var(--text-2)" },
  ".cm-fence": { backgroundColor: "var(--tile)", fontFamily: "var(--mono)", fontSize: "0.88em", padding: "0 12px" },
  ".cm-table": { fontFamily: "var(--mono)", fontSize: "0.88em" },
  ".cm-bullet": { display: "inline-block", width: "1.1em", color: "var(--text-2)" },
  ".cm-task": { margin: "0 6px 0 0", verticalAlign: "-1px", accentColor: "var(--chrome)", cursor: "pointer" },
  ".cm-rule": { display: "block", margin: "0.7em 0", borderTop: "1px solid var(--line)" },
});

/** What the editor holds besides the note: undo, the keys, the look and the live preview. `openLink` is called for a Ctrl- or ⌘-click on a link. */
export function liveEditor(options: { placeholder: string; openLink: (href: string) => void; label: string }) {
  return [
    history(),
    markdown,
    preview,
    theme,
    EditorView.lineWrapping,
    placeholder(options.placeholder),
    EditorView.contentAttributes.of({ spellcheck: "true", "aria-label": options.label, "aria-multiline": "true" }),
    Prec.high(keymap.of([{ key: "Enter", run: continueList }])),
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.domEventHandlers({
      mousedown(event) {
        const link = (event.target as HTMLElement).closest<HTMLElement>(".cm-link");
        if (!link?.dataset.href || !(event.ctrlKey || event.metaKey)) return false;
        event.preventDefault(); // a click must not navigate the notes window itself
        options.openLink(link.dataset.href);
        return true;
      },
    }),
  ];
}

export { EditorState, EditorView };
