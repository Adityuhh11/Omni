import { Events } from "@wailsio/runtime";
import { Service as Settings } from "../../bindings/omni/internal/settingsui";
import { isMac } from "./platform";

/** The themes every system has, in the order the picker lists them; `plate` and `text` are what its swatch shows. Same names as `config.Themes`. */
const COMMON_THEMES = [
  { id: "black", label: "Black", plate: "#000000", text: "#eceef2" },
  { id: "midnight", label: "Midnight", plate: "#0a0f1e", text: "#e6ebf7" },
  { id: "forest", label: "Forest", plate: "#09140f", text: "#e4efe8" },
  { id: "burgundy", label: "Burgundy", plate: "#1a0609", text: "#f3e4e6" },
  { id: "cream", label: "Cream", plate: "#f6f0e2", text: "#2a2620" },
  { id: "blush", label: "Blush", plate: "#f8e8e6", text: "#33201f" },
  { id: "coral", label: "Coral", plate: "#f1c9c2", text: "#3a1512" },
  { id: "glass", label: "Glass", plate: "linear-gradient(135deg, #5b6170, #1a1c24 55%, #3a3550)", text: "#ffffff" },
] as const;

/** The accents, as the picker shows them. Same names as `config.Accents`; "none" is the chrome. */
export const ACCENTS = [
  { id: "none", label: "Chrome", swatch: "#e6e8ec" },
  { id: "blue", label: "Blue", swatch: "#7aa7ff" },
  { id: "violet", label: "Violet", swatch: "#b49cff" },
  { id: "green", label: "Green", swatch: "#6fd3a0" },
  { id: "amber", label: "Amber", swatch: "#f2b862" },
  { id: "rose", label: "Rose", swatch: "#f28fae" },
] as const;

/** Glass 2 needs the Mac's blur behind the window (panel.Glass), so only a Mac lists it. */
const GLASS2 = { id: "glass2", label: "Glass 2", plate: "#1c1d21", text: "#ffffff" } as const;

/** The themes this system offers, in the order the picker lists them. */
export const THEMES: readonly { id: string; label: string; plate: string; text: string }[] = isMac ? [...COMMON_THEMES, GLASS2] : COMMON_THEMES;

/** The Glass 2 bar's accents, offered beside its free colour; the first is a fresh install's (`config.DefaultBarAccent`). */
export const BAR_ACCENTS = [
  { hex: "#e5484d", label: "Red" },
  { hex: "#f2601d", label: "Ember" },
  { hex: "#f5d90a", label: "Yellow" },
  { hex: "#46a758", label: "Green" },
  { hex: "#5bb8d0", label: "Teal" },
  { hex: "#3e63dd", label: "Blue" },
  { hex: "#8e4ec6", label: "Violet" },
  { hex: "#d6409f", label: "Pink" },
] as const;
const HEX = /^#[0-9a-f]{6}$/i;

/** What every window wears; the launcher's config.UI. */
export type Look = { Theme: string; Ring: boolean; Accent: string; Lights: boolean; BarAccent: string; BarBorder: boolean };

const KEY = "omni.look";
const DEFAULT: Look = { Theme: "black", Ring: true, Accent: "none", Lights: false, BarAccent: BAR_ACCENTS[0].hex, BarBorder: true };

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
    Lights: d.lights === "on",
    BarAccent: d.barAccent ?? DEFAULT.BarAccent,
    BarBorder: d.barBorder !== "false",
  };
}

/** Paints the page in a look and remembers it, so the next window starts in it without waiting for the launcher. Unknown names
 *  keep what the page had. */
export function applyLook(next: Partial<Look>) {
  const before = currentLook();
  const look: Look = {
    Theme: THEMES.some((t) => t.id === next.Theme) ? next.Theme! : before.Theme,
    Ring: next.Ring ?? before.Ring,
    Accent: ACCENTS.some((a) => a.id === next.Accent) ? next.Accent! : before.Accent,
    Lights: next.Lights ?? before.Lights,
    BarAccent: next.BarAccent && HEX.test(next.BarAccent) ? next.BarAccent.toLowerCase() : before.BarAccent,
    BarBorder: next.BarBorder ?? before.BarBorder,
  };
  const d = document.documentElement.dataset;
  d.theme = look.Theme;
  d.ring = look.Ring ? "on" : "off";
  d.accent = look.Accent;
  d.lights = look.Lights ? "on" : "off";
  d.barAccent = look.BarAccent;
  d.barBorder = String(look.BarBorder);
  document.documentElement.style.setProperty("--bar-accent", look.BarAccent);
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
  applyLook(saved);
  Settings.Look().then(applyLook, (err) => console.warn("look:", err));
  Events.On("ui:look", (e) => applyLook(e.data as Look));
}
