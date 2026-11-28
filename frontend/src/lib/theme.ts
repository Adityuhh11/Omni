import { Events } from "@wailsio/runtime";
import { Service as Settings } from "../../bindings/omni/internal/settingsui";
import { isMac } from "./platform";

/** The themes, in the order the picker lists them; `plate` and `text` are what its swatch shows. Same names as `config.Themes`. */
export const THEMES: readonly { id: string; label: string; plate: string; text: string }[] = [
  { id: "black", label: "Black", plate: "#000000", text: "#eceef2" },
  { id: "midnight", label: "Midnight", plate: "#0a0f1e", text: "#e6ebf7" },
  { id: "forest", label: "Forest", plate: "#09140f", text: "#e4efe8" },
  { id: "burgundy", label: "Burgundy", plate: "#1a0609", text: "#f3e4e6" },
  { id: "cream", label: "Cream", plate: "#f6f0e2", text: "#2a2620" },
  { id: "blush", label: "Blush", plate: "#f8e8e6", text: "#33201f" },
  { id: "coral", label: "Coral", plate: "#f1c9c2", text: "#3a1512" },
  { id: "glass", label: "Glass", plate: "linear-gradient(135deg, #5b6170, #1a1c24 55%, #3a3550)", text: "#ffffff" },
];

/** The accents, as the picker shows them, the default first. Same names as `config.Accents`; "none" is the chrome. A person may also pick any
 *  colour, which is kept as "#rrggbb". */
export const ACCENTS = [
  { id: "ember", label: "Ember", swatch: "#f2601d" },
  { id: "none", label: "Chrome", swatch: "#e6e8ec" },
  { id: "blue", label: "Blue", swatch: "#7aa7ff" },
  { id: "violet", label: "Violet", swatch: "#b49cff" },
  { id: "green", label: "Green", swatch: "#6fd3a0" },
  { id: "amber", label: "Amber", swatch: "#f2b862" },
  { id: "rose", label: "Rose", swatch: "#f28fae" },
] as const;

/** Any accent colour a person picks is written like this. */
const HEX = /^#[0-9a-f]{6}$/i;

/** What every window wears; the launcher's config.UI. */
export type Look = { Theme: string; Ring: boolean; Accent: string; BarBorder: boolean; Liquid: boolean };

const KEY = "omni.look";
const DEFAULT: Look = { Theme: "black", Ring: true, Accent: ACCENTS[0].id, BarBorder: true, Liquid: true };

const listeners = new Set<(look: Look) => void>();

/** Calls `fn` with the look whenever the page's theme, ring or accent changes; returns what stops it. */
export function onLook(fn: (look: Look) => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** The look the page wears now. */
export function currentLook(): Look {
  const d = document.documentElement.dataset;
  return {
    Theme: d.theme ?? DEFAULT.Theme,
    Ring: d.ring !== "off",
    Accent: d.accent ?? DEFAULT.Accent,
    BarBorder: d.barBorder !== "false",
    Liquid: d.liquid !== "false",
  };
}

/** A colour of the person's own sets the accent's three colours itself, where a named accent has them in theme.css: the text on a filled
 *  button is dark or white, whichever reads better, and the hover fill is the colour a little lighter (darker on a light theme). */
function paintAccent(accent: string) {
  const style = document.documentElement.style;
  if (!HEX.test(accent)) {
    for (const name of ["--chrome", "--chrome-hot", "--on-chrome"]) style.removeProperty(name);
    return;
  }
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(accent.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  const light = getComputedStyle(document.documentElement).colorScheme === "light";
  style.setProperty("--chrome", accent);
  style.setProperty("--chrome-hot", `color-mix(in srgb, ${accent} 70%, ${light ? "black" : "white"})`);
  style.setProperty("--on-chrome", 0.2126 * r + 0.7152 * g + 0.0722 * b > 0.35 ? "#0b0c0e" : "#fff");
}

/** Paints the page in a look and remembers it, so the next window starts in it without waiting for the launcher. Unknown names
 *  keep what the page had. */
export function applyLook(next: Partial<Look>) {
  const before = currentLook();
  const look: Look = {
    Theme: THEMES.some((t) => t.id === next.Theme) ? next.Theme! : before.Theme,
    Ring: next.Ring ?? before.Ring,
    Accent: ACCENTS.some((a) => a.id === next.Accent) ? next.Accent! : next.Accent && HEX.test(next.Accent) ? next.Accent.toLowerCase() : before.Accent,
    BarBorder: next.BarBorder ?? before.BarBorder,
    Liquid: next.Liquid ?? before.Liquid,
  };
  const d = document.documentElement.dataset;
  d.theme = look.Theme;
  d.system = isMac ? "mac" : "windows"; // Glass's blur differs: a Mac's is tinted, Windows' is not (theme.css)
  d.ring = look.Ring ? "on" : "off";
  d.accent = look.Accent; // a colour of the person's own stays as "#rrggbb": theme.css matches it by its "#"
  d.barBorder = String(look.BarBorder);
  d.liquid = String(look.Liquid);
  paintAccent(look.Accent);
  if ((Object.keys(look) as (keyof Look)[]).some((k) => look[k] !== before[k])) listeners.forEach((fn) => fn(look));
  try {
    localStorage.setItem(KEY, JSON.stringify(look));
  } catch {
    // no storage in this web view: the launcher is asked at every start instead
  }
}

/** Starts the page in the last look it knew, asks the launcher for the real one and follows changes made in Settings. */
export function followLook() {
  let saved: Partial<Look> = DEFAULT;
  try {
    // Before the ring and the accent, only the theme was kept, under its own name.
    const kept = localStorage.getItem(KEY);
    saved = kept ? { ...DEFAULT, ...JSON.parse(kept) } : { ...DEFAULT, Theme: localStorage.getItem("omni.theme") ?? DEFAULT.Theme };
  } catch {
    // nothing kept, or not readable: the default until the launcher answers
  }
  if (saved.Theme === "glass2") saved.Theme = "glass"; // the second Glass is the only one now
  applyLook(saved);
  Settings.Look().then(applyLook, (err) => console.warn("look:", err));
  Events.On("ui:look", (e) => applyLook(e.data as Look));
}
