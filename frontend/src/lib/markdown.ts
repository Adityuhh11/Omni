/**
 * A small markdown renderer for notes, Omni AI's answers and calendar events: headings, paragraphs, quotes, lists (also numbered,
 * nested and task lists), tables, code and code blocks, rules, and bold, italic, strike-through, code, links. Everything in
 * the text is escaped first and only the tags written here are added, so a note can never put script or markup into the page. Links are limited to http, https and mailto.
 */

const escapes: Record<string, string> = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
const escape = (text: string) => text.replace(/[&<>"']/g, (c) => escapes[c]);

/** Inline marks inside one already-escaped line. Code spans are lifted out first so their content is left alone. */
function inline(escaped: string): string {
  if (escaped.length > MAX_INLINE) return escaped; // a pasted wall of text is shown as it is: the patterns below are not meant for it
  // Code spans and links are lifted out first and put back last, so emphasis marks inside them (a URL with __x__ in it) are never touched.
  const spans: string[] = [];
  const lift = (html: string) => `\u0000${spans.push(html) - 1}\u0000`;
  let text = escaped.replace(/`([^`]+)`/g, (_, code) => lift(`<code>${code}</code>`));
  text = text.replace(/\[([^\]]+)\]\(((?:https?:\/\/|mailto:)(?:(?!&(?:quot|#39|lt|gt);)[^\s)])+)\)/gi, (_, label, url) => lift(`<a href="${url}">${label}</a>`));
  text = text.replace(/(^|[\s(])(https?:\/\/(?:(?!&(?:quot|#39|lt|gt);)[^\s<)])+(?<![.,;:!?]))/gi, (_, before, url) => `${before}${lift(`<a href="${url}">${url}</a>`)}`);
  text = text.replace(/\*\*([^*]+)\*\*|__([^_]+)__/g, (_, a, b) => `<strong>${a ?? b}</strong>`);
  text = text.replace(/(^|[^\w*])\*([^*\s][^*]*)\*(?!\w)|(^|[^\w_])_([^_\s][^_]*)_(?!\w)/g, (_, p1, a, p2, b) =>
    a !== undefined ? `${p1}<em>${a}</em>` : `${p2}<em>${b}</em>`,
  );
  text = text.replace(/~~([^~]+)~~/g, "<del>$1</del>");
  const put = (t: string) => t.replace(/\u0000(\d+)\u0000/g, (_, i) => spans[Number(i)] ?? "");
  return put(put(text)); // a link's words can hold a code span, which is itself a placeholder
}

const MAX_INLINE = 4000; // characters of one line that get emphasis and links
const MAX_DEPTH = 8; // levels of quotes and lists that are nested; deeper ones are shown as text

const heading = /^(#{1,6})\s+(.*)$/;
const rule = /^\s*([-*_])(\s*\1){2,}\s*$/;
const listItem = /^(\s*)([-*+]|\d{1,9}[.)])\s+(.*)$/;
const task = /^\[([ xX])\]\s+(.*)$/;
const quote = /^>\s?(.*)$/;
const fence = /^(\s*)```/;
const tableRule = /^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?\s*$/;

const indentOf = (line: string) => line.length - line.trimStart().length;
const isBlank = (line: string) => line.trim() === "";
const isTable = (lines: string[], i: number) => lines[i].includes("|") && i + 1 < lines.length && tableRule.test(lines[i + 1]);

/** A table row's cells: the outer pipes dropped, split on the pipes that are not escaped. */
function cells(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/, "")
    .replace(/(?<!\\)\|$/, "")
    .split(/(?<!\\)\|/)
    .map((c) => inline(escape(c.trim().replaceAll("\\|", "|"))));
}

/** Whether a line starts a block of its own, which ends a paragraph. */
function startsBlock(lines: string[], i: number, depth: number): boolean {
  const line = lines[i];
  return isBlank(line) || fence.test(line) || heading.test(line) || rule.test(line) || listItem.test(line) || isTable(lines, i) || (quote.test(line) && depth < MAX_DEPTH);
}

/** Renders markdown to HTML. A task's box carries `data-line`, the line of the source it came from (see toggleTask). */
export function render(markdown: string, firstLine = 0, depth = 0): string {
  const lines = markdown.replace(/\r\n?/g, "\n").replaceAll("\u0000", "").split("\n");
  const out: string[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (isBlank(line)) {
      i++;
    } else if (fence.test(line)) {
      // A fence indented under a list item is still a fence; its lines lose that indent.
      const indent = line.match(fence)![1].length;
      const code: string[] = [];
      i++;
      while (i < lines.length && !fence.test(lines[i])) {
        const l = lines[i++];
        code.push(l.slice(Math.min(indent, indentOf(l))));
      }
      i++; // the closing fence (a missing one ends the block at the end of the note)
      out.push(`<pre><code>${escape(code.join("\n"))}</code></pre>`);
    } else if (heading.test(line)) {
      const [, hashes, text] = line.match(heading)!;
      out.push(`<h${hashes.length}>${inline(escape(text.trim()))}</h${hashes.length}>`);
      i++;
    } else if (rule.test(line)) {
      out.push("<hr>");
      i++;
    } else if (isTable(lines, i)) {
      const head = cells(line);
      const rows: string[] = [];
      i += 2;
      while (i < lines.length && !isBlank(lines[i]) && lines[i].includes("|")) {
        const row = cells(lines[i++]);
        rows.push(`<tr>${head.map((_, c) => `<td>${row[c] ?? ""}</td>`).join("")}</tr>`);
      }
      out.push(`<div class="md-table"><table><thead><tr>${head.map((h) => `<th>${h}</th>`).join("")}</tr></thead><tbody>${rows.join("")}</tbody></table></div>`);
    } else if (quote.test(line) && depth < MAX_DEPTH) {
      const start = i;
      const body: string[] = [];
      while (i < lines.length && quote.test(lines[i])) body.push(lines[i++].match(quote)![1]);
      // The quoted lines keep their place in the note, so a task inside a quote toggles its own line (see toggleTask).
      out.push(`<blockquote>${render(body.join("\n"), firstLine + start, depth + 1)}</blockquote>`);
    } else if (listItem.test(line)) {
      i = list(lines, i, firstLine, depth, out);
    } else {
      const para: string[] = [];
      do para.push(inline(escape(lines[i++])));
      while (i < lines.length && !startsBlock(lines, i, depth));
      out.push(`<p>${para.join("<br>")}</p>`);
    }
  }
  return out.join("\n");
}

/**
 * One list from line i: its items are the lines with its marker at its indent, and what is indented under an item (more
 * text, a nested list, a code block) belongs to that item, also across blank lines. A numbered list keeps the number it
 * starts at. Returns the line after the list.
 */
function list(lines: string[], i: number, firstLine: number, depth: number, out: string[]): number {
  const [, lead, marker] = lines[i].match(listItem)!;
  const indent = lead.length;
  const ordered = /\d/.test(marker);
  const sameKind = (l: string) => {
    const m = l.match(listItem);
    return m !== null && m[1].length === indent && /\d/.test(m[2]) === ordered;
  };
  const items: string[] = [];
  while (i < lines.length && sameKind(lines[i])) {
    const text = lines[i].match(listItem)![3].trimEnd();
    const at = i++;
    const bodyStart = i;
    while (i < lines.length) {
      if (!isBlank(lines[i])) {
        if (indentOf(lines[i]) <= indent) break;
        i++;
        continue;
      }
      let next = i;
      while (next < lines.length && isBlank(lines[next])) next++;
      if (next === lines.length || indentOf(lines[next]) <= indent) break;
      i = next;
    }
    const body = lines.slice(bodyStart, i);
    const t = !ordered ? text.match(task) : null;
    const head = t ? `<input type="checkbox" data-line="${firstLine + at}"${t[1] !== " " ? " checked" : ""}> <span>${inline(escape(t[2]))}</span>` : inline(escape(text));
    let rest = "";
    if (body.length) {
      const cut = body.reduce((least, l) => (isBlank(l) ? least : Math.min(least, indentOf(l))), Infinity);
      rest =
        depth < MAX_DEPTH
          ? render(body.map((l) => l.slice(cut)).join("\n"), firstLine + bodyStart, depth + 1)
          : `<p>${body.filter((l) => !isBlank(l)).map((l) => inline(escape(l.trim()))).join("<br>")}</p>`;
    }
    items.push(`<li${t ? ` class="task${t[1] !== " " ? " done" : ""}"` : ""}>${head}${rest}</li>`);
    // Blank lines between two items of the same list do not end it.
    let next = i;
    while (next < lines.length && isBlank(lines[next])) next++;
    if (next < lines.length && sameKind(lines[next])) i = next;
  }
  const start = ordered ? parseInt(marker, 10) : 1;
  out.push(ordered ? `<ol${start !== 1 ? ` start="${start}"` : ""}>${items.join("")}</ol>` : `<ul>${items.join("")}</ul>`);
  return i;
}

/** The note with the task on this line (0-based) ticked if it was open, or unticked if it was done. Other lines are left alone. */
export function toggleTask(markdown: string, line: number): string {
  const lines = markdown.split("\n");
  const at = lines[line];
  if (at === undefined) return markdown;
  const toggled = at.replace(/^(\s*(?:>\s?)*[-*+]\s+)\[([ xX])\]/, (_, before, mark) => `${before}[${mark === " " ? "x" : " "}]`);
  if (toggled === at) return markdown;
  lines[line] = toggled;
  return lines.join("\n");
}
