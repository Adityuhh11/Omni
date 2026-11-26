import type { Item } from "../../../bindings/omni/internal/palette";
import type { Row } from "../../../bindings/omni/internal/search";

/** One thing the welcome screen's bar types out: what is typed, the rows the real bar would answer with, and a line saying why it matters. */
export interface Scene {
  id: string;
  /** The chip that jumps to it. */
  label: string;
  icon: "app" | "web" | "calc" | "clip" | "note" | "timer" | "search";
  query: string;
  caption: string;
  /** Rows drawn by the bar's own row components; the first is the selected one. */
  items?: Item[];
  files?: Row[];
}

const item = (Kind: string, Title: string, Subtitle: string): Item => ({ ID: `${Kind}:demo`, Kind, Title, Subtitle, Icon: "", Actions: null, Fill: "" });

const file = (Name: string, Folder: string, Kind: string, Location: string, Snippet: string): Row => ({
  Path: "",
  Name,
  Folder,
  Kind,
  Line: 0,
  Timed: false,
  Seconds: 0,
  Score: 1,
  Matches: [{ Location, Snippet, Line: 0, Text: Snippet, Timed: false, Seconds: 0 }],
});

export const SCENES: Scene[] = [
  {
    id: "apps",
    label: "Apps",
    icon: "app",
    query: "vsc",
    caption: "A few letters open any app. The ones you use most come first.",
    items: [item("app", "Visual Studio Code", "Application"), item("app", "Visual Studio Installer", "Application"), item("web", "Search the web for “vsc”", "Web")],
  },
  {
    id: "web",
    label: "Web",
    icon: "web",
    query: "gh rust lifetimes",
    caption: "Quicklinks jump straight to a site's own search. Add your own.",
    items: [item("quicklink", "Search GitHub for “rust lifetimes”", "Quicklink"), item("web", "Search the web for “gh rust lifetimes”", "Web")],
  },
  {
    id: "calc",
    label: "Maths",
    icon: "calc",
    query: "12*7+3",
    caption: "Sums, percentages and unit conversions, answered as you type.",
    items: [item("calc", "87", "Enter copies the answer"), item("web", "Search the web for “12*7+3”", "Web")],
  },
  {
    id: "clip",
    label: "Clipboard",
    icon: "clip",
    query: "clip:",
    caption: "Everything you copied lately, newest first. Enter pastes it where you were. Off until you switch it on.",
    items: [item("clip", "Invoice #2041, due 14 March", "Clipboard"), item("clip", "https://github.com/rust-lang/rust", "Clipboard"), item("clip", "Meet at 3 in room 4", "Clipboard")],
  },
  {
    id: "note",
    label: "Notes",
    icon: "note",
    query: "note: buy milk",
    caption: "Jot something down without leaving what you were doing. Notes are plain markdown files.",
    items: [item("note", "Make a note: buy milk", "Note"), item("note", "Groceries", "Note")],
  },
  {
    id: "timer",
    label: "Timer",
    icon: "timer",
    query: "timer 25",
    caption: "Timers, pomodoro and focus periods, in a small window at the edge of your screen.",
    items: [item("timer", "Start a 25 minute timer", "Timer"), item("timer", "Start a pomodoro", "Timer")],
  },
  {
    id: "files",
    label: "Files",
    icon: "search",
    query: "the diagram with three boxes",
    caption: "Optional: find files by what they say or show, not what they are called.",
    files: [file("System design.pdf", "Documents/Work", "pdf", "page 4", "client, API and queue: the three boxes in the diagram")],
    items: [item("web", "Search the web for “the diagram with three boxes”", "Web")],
  },
];
