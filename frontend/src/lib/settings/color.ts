/** Hue 0-360, saturation and value 0-1: the picker's way of seeing a colour. */
export type Hsv = { h: number; s: number; v: number };

const clamp = (n: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, n));

/** What a person typed or pasted, as "#rrggbb": "f2601d", "#F2601D" and "#f60" all work. Null for anything else. */
export function normalizeHex(text: string): string | null {
  const t = text.trim().replace(/^#/, "").toLowerCase();
  if (/^[0-9a-f]{3}$/.test(t)) return "#" + [...t].map((c) => c + c).join("");
  return /^[0-9a-f]{6}$/.test(t) ? "#" + t : null;
}

export function hexToHsv(hex: string): Hsv {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const max = Math.max(r, g, b);
  const d = max - Math.min(r, g, b);
  let h = 0;
  if (d > 0) {
    if (max === r) h = ((g - b) / d) % 6;
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h = (h * 60 + 360) % 360;
  }
  return { h, s: max === 0 ? 0 : d / max, v: max };
}

export function hsvToHex({ h, s, v }: Hsv): string {
  const channel = (n: number) => {
    const k = (n + h / 60) % 6;
    return v - v * s * clamp(Math.min(k, 4 - k), 0, 1);
  };
  return "#" + [channel(5), channel(3), channel(1)].map((c) => Math.round(c * 255).toString(16).padStart(2, "0")).join("");
}
