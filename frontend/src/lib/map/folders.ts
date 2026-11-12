/**
 * The map colours its dots by folder: the eight categorical hues of the dark palette (validated on the map's night: every
 * adjacent pair stays apart for colour-blind eyes), in a fixed order. A ninth folder and on, and a folder no longer
 * searched, are grey: "other", never a made-up hue. Identity is never colour alone: the legend and the hover card name it.
 */

export const SLOTS = ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"];
const OTHER = "#6b6e76";

/** The point colours (rgba, 0-1) for each point's folder: solid, or alpha 0 for a hidden folder's points. */
export function colorsFor(source: Uint16Array, roots: string[], hidden: Set<number>): Float32Array {
  const palette = roots.map((root, i) => rgb(i < SLOTS.length && root ? SLOTS[i] : OTHER));
  const out = new Float32Array(source.length * 4);
  for (let i = 0; i < source.length; i++) {
    const [r, g, b] = palette[source[i]] ?? rgb(OTHER);
    out.set([r, g, b, hidden.has(source[i]) ? 0 : 1], i * 4);
  }
  return out;
}

/** The last part of a folder's path, as the legend shows it. */
export function folderName(root: string): string {
  if (!root) return "No longer searched";
  const parts = root.split(/[\\/]+/).filter(Boolean);
  return parts[parts.length - 1] ?? root;
}

function rgb(hex: string): [number, number, number] {
  const n = parseInt(hex.slice(1), 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}
