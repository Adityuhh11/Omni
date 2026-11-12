/**
 * The map colours its dots by folder or by file type: the eight categorical hues of the dark palette (validated on the
 * map's night: every adjacent pair stays apart for colour-blind eyes), in a fixed order. A ninth group and on, "other",
 * and a folder no longer searched are grey, never a made-up hue. Identity is never colour alone: the legend and the hover
 * card name it. By age it is a ramp instead, from the oldest file to the newest.
 */

export const SLOTS = ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"];
export const OTHER = "#6b6e76";
// The upper part of viridis: even steps of lightness, readable on the night and for colour-blind eyes.
export const AGE_RAMP = ["#3b528b", "#21918c", "#5ec962", "#fde725"];

/** The colour of group index named name (a folder's path, a file type; "" is a folder no longer searched). */
export function slotColor(index: number, name: string): string {
  return index < SLOTS.length && name && name !== "other" ? SLOTS[index] : OTHER;
}

/** The point colours (rgba, 0-1) for each point's group: solid, or alpha 0 for a hidden group's points. */
export function colorsFor(group: Uint16Array, names: string[], hidden: Set<number>): Float32Array {
  const palette = names.map((name, i) => rgb(slotColor(i, name)));
  const out = new Float32Array(group.length * 4);
  for (let i = 0; i < group.length; i++) {
    const [r, g, b] = palette[group[i]] ?? rgb(OTHER);
    out.set([r, g, b, hidden.has(group[i]) ? 0 : 1], i * 4);
  }
  return out;
}

/**
 * The point colours by age, and the dates at the ramp's ends and middle. Points are placed on the ramp by rank, not by
 * date, so a few very old files do not squeeze everything else into one colour. An unknown date (0) is grey.
 */
export function ageColors(modified: Uint32Array): { colors: Float32Array; cuts: [number, number, number] } {
  const known = Array.from(modified).filter((t) => t > 0).sort((a, b) => a - b);
  const out = new Float32Array(modified.length * 4);
  const stops = AGE_RAMP.map(rgb);
  const at = (q: number) => known[Math.min(known.length - 1, Math.floor(q * known.length))] ?? 0;
  for (let i = 0; i < modified.length; i++) {
    const t = modified[i];
    if (!t) {
      out.set([...rgb(OTHER), 1], i * 4);
      continue;
    }
    const q = known.length > 1 ? rankOf(known, t) / (known.length - 1) : 1;
    const x = q * (stops.length - 1);
    const lo = Math.min(stops.length - 2, Math.floor(x));
    const f = x - lo;
    const [a, b] = [stops[lo], stops[lo + 1]];
    out.set([a[0] + (b[0] - a[0]) * f, a[1] + (b[1] - a[1]) * f, a[2] + (b[2] - a[2]) * f, 1], i * 4);
  }
  return { colors: out, cuts: [at(0), at(0.5), at(1)] };
}

/** The last part of a folder's path, as the legend shows it. */
export function folderName(root: string): string {
  if (!root) return "No longer searched";
  const parts = root.split(/[\\/]+/).filter(Boolean);
  return parts[parts.length - 1] ?? root;
}

// The first place of t in sorted (binary search): equal dates share a colour.
function rankOf(sorted: number[], t: number): number {
  let lo = 0;
  let hi = sorted.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (sorted[mid] < t) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

function rgb(hex: string): [number, number, number] {
  const n = parseInt(hex.slice(1), 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}
