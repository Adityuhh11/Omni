<script lang="ts">
  import { onMount, tick } from "svelte";
  import { fade } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import { Clipboard, Events } from "@wailsio/runtime";
  import { Service as Files } from "../bindings/omni/internal/files";
  import { Service as Palette } from "../bindings/omni/internal/palette";
  import type { Item as Entry } from "../bindings/omni/internal/palette";
  import { Service as Panel } from "../bindings/omni/internal/panel";
  import { Service as Search } from "../bindings/omni/internal/search";
  import { Service as SettingsWindow } from "../bindings/omni/internal/settingsui";
  import type { Results } from "../bindings/omni/internal/search";
  import { Service as Status } from "../bindings/omni/internal/status";
  import type { Info } from "../bindings/omni/internal/status";
  import ActionPanel from "./lib/ActionPanel.svelte";
  import EntryRow from "./lib/EntryRow.svelte";
  import Hints from "./lib/Hints.svelte";
  import SearchGlyph from "./lib/SearchGlyph.svelte";
  import LiquidRing from "./lib/LiquidRing.svelte";
  import { glass, glassPane } from "./lib/glass";
  import { showUnexpectedIn } from "./lib/oops";
  import Loader from "./lib/Loader.svelte";
  import { nextMetal, ringsFor } from "./lib/palettes";
  import { currentLook, onLook } from "./lib/theme";
  import ResultRow from "./lib/ResultRow.svelte";
  import { actionsFor, available, type Way } from "./lib/actions";
  import { filePathOf, flatten, keyOf, openLine, pathOf, pinIdOf, step, tagIdOf, timeOf, type Item } from "./lib/flatten";
  import { keyCommand } from "./lib/keys";
  import { isMac } from "./lib/platform";
  import { asksForFiles, cycle, escapeStep, label, type Mode } from "./lib/modes";
  import { notesFor } from "./lib/notes";
  import TagPicker from "./lib/TagPicker.svelte";

  const DEBOUNCE_MS = 120;
  const WARMING_RETRY_MS = 1500;
  const RING = 2.5; // px of metal around the card
  const MARGIN_TOP = 12; // clear window above and below the card, for its shadow
  const MARGIN_BOTTOM = 40;
  const BAR_HEIGHT = 60;
  const MORPH_MS = 240; // how long the card takes to grow or shrink (keep in step with the CSS)

  let info = $state<Info | null>(null);
  let results = $state<Results | null>(null);
  let query = $state("");
  let answered = $state(""); // the query `results` answers, so emphasis does not flicker while typing
  let selected = $state(0);
  let expanded = $state(new Set<string>());
  let problem = $state("");
  let active = $state(false); // the panel has the keyboard
  let theme = $state(currentLook().Theme);
  let ring = $state(currentLook().Ring); // the metal ring, or a plain edge
  let metal = $state(0); // which of the theme's rings the bar wears; changes whenever the panel closes
  const rings = $derived(ringsFor(theme));
  // Glass has no metal: its edge is a hairline in the bar's accent. The other themes wear the ring unless it is turned off.
  const ringed = $derived(theme !== "glass" && ring);
  let pending = $state(false); // a search is in flight
  let typing = $state(false); // a key was pressed in the last moment
  let target = $state<Item | null>(null); // the row the open action panel acts on (null: panel closed)
  let mode = $state<Mode>("all"); // the chip: what a search covers, cycled with Tab
  let effective = $state<Mode>("all"); // the mode of the text once a prefix such as "app:" is taken off
  let entries = $state<Entry[]>([]); // the instant answers (apps): they arrive in a millisecond, before the files
  let picker = $state<{ id: string; title: string } | null>(null); // the thing the tag picker (Ctrl+T) is open on
  let tagsByID = $state<Record<string, string[] | null | undefined>>({}); // the tags of the rows on screen, shown as chips
  let answerText = $state(""); // the text the instant answer searched, with any prefix taken off
  let answerNote = $state(""); // why the Clipboard mode has nothing to list (it is off, nothing matches), if it has not
  let pins = $state(new Set<string>()); // the ids of everything pinned, to mark the rows
  let pinnedRows = $state<Entry[]>([]); // the pinned things that still exist, in the order pinned: the bar's empty state
  let quick = $state<Entry[]>([]); // the quick-open row under the empty bar: number keys 1 to its length open one
  const showQuick = $derived(quick.length > 0); // beside the bar while it is open, so the bar does not change width as you type
  // The tiles are as tall as the bar beside them (Glass's bar is the slim one: its .bar is 50 px, see the style below).
  const quickSize = $derived((theme === "glass" ? 50 : BAR_HEIGHT) + RING * 2);
  let cardWidth = $state(0);
  let keying = $state(false); // an arrow key moved the selection a moment ago: the list lifts the selected row and dims the rest
  let keyingTimer: ReturnType<typeof setTimeout> | undefined;
  const KEYING_MS = 900;
  let instant = 0;
  let input: HTMLInputElement;
  let content: HTMLElement;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let token = 0;

  // Beside apps, files without a confident match are probably noise, so they are not shown at all.
  const local = $derived(entries.filter((e) => e.Kind !== "web")); // answers about your own machine
  const fileRows = $derived(results && (results.Confident || local.length === 0) ? (results.Rows ?? []) : []);
  const showingPins = $derived(query.trim() === "" && mode === "all"); // nothing typed: the bar lists what is pinned
  const items = $derived<Item[]>(flatten(fileRows, expanded, entries, showingPins, info?.GroupResults ?? false));
  const current = $derived<Item | undefined>(items[selected]);
  // The program a moment in a recording opens in ("VLC"), or "" when none can be asked: asked when such a row is selected, so
  // a player installed meanwhile counts, and the hint says which Enter will do.
  let player = $state("");
  $effect(() => {
    if (current && current.kind !== "entry" && timeOf(current) !== null) Files.SeekPlayer().then((name) => (player = name));
  });
  const firstApp = $derived(items.findIndex((i) => i.kind === "entry" && i.entry.Kind === "app"));
  const firstFile = $derived(items.findIndex((i) => i.kind === "row"));
  const sections = $derived(firstApp >= 0 && firstFile >= 0); // two kinds of rows: label them
  const searchWhat = $derived(
    effective === "files" ? "Search your files" : info?.FilesInBar ? "Search apps, files and the web" : "Search apps and the web",
  );
  const chip = $derived(results?.Scope || label[effective]);

  // The tags of the rows on screen, for their chips.
  $effect(() => {
    const ids = items.map(tagIdOf).filter((id): id is string => id !== null);
    let fresh = true;
    if (ids.length === 0) tagsByID = {};
    else Palette.TagsOf(ids).then((found) => fresh && (tagsByID = found ?? {})); // Go sends null for none
    return () => (fresh = false);
  });
  const tagsOf = (item: Item) => tagsByID[tagIdOf(item) ?? ""] ?? [];

  // What the page waits for, in a word or two, shown as the loader in the bar.
  const waiting = $derived.by(() => {
    if (info && !["ready", "failed", "off", ""].includes(info.Engine)) return "Warming up";
    if (info?.IndexState === "indexing") return "Indexing";
    return "";
  });

  const notes = $derived(
    notesFor({
      info,
      problem,
      query,
      effective,
      entries: entries.length,
      local: local.length,
      answerNote,
      results,
      fileRows: fileRows.length,
      answered,
    }),
  );
  const showPanel = $derived(notes.length > 0 || items.length > 0);

  // The card is exactly as tall as what it holds, plus its ring; the window around it is taller by the
  // shadow margins. The card animates to its new height in CSS; the window jumps to the larger of the
  // old and new height at once (so nothing is clipped mid-motion) and shrinks after the motion.
  let contentHeight = $state(0);
  const cardHeight = $derived(contentHeight > 0 ? Math.ceil(contentHeight + RING * 2) : BAR_HEIGHT + RING * 2);
  const windowHeight = $derived(cardHeight + MARGIN_TOP + MARGIN_BOTTOM);
  let appliedHeight = 0;
  let shrinkTimer: ReturnType<typeof setTimeout> | undefined;

  $effect(() => {
    const height = windowHeight;
    clearTimeout(shrinkTimer);
    if (height >= appliedHeight) resizeWindow(height);
    else shrinkTimer = setTimeout(() => resizeWindow(height), MORPH_MS + 40);
  });

  // Glass has a blur of the desktop under the card, put there by the launcher as a view of its own: it takes the card's place and
  // size, and moves over the same time and easing as the card does.
  const SIDE = 20; // clear window to the left and right of the card
  $effect(() => {
    if (theme !== "glass") {
      Panel.Glass(false, 0, 0, 0, 0, 0, 0);
      return;
    }
    const radius = parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--radius")) || 32;
    Panel.Glass(true, SIDE, MARGIN_TOP, cardWidth || window.innerWidth - SIDE * 2, cardHeight, radius, MORPH_MS);
  });

  function resizeWindow(height: number) {
    appliedHeight = height;
    Panel.Resize(height);
  }

  $effect(() => {
    document.documentElement.dataset.active = String(active);
  });

  // One highlight under the selected row that slides when the selection moves, instead of each row lighting up on its own.
  // A new answer puts it in place at once: sliding from where the old list's row was would point at nothing.
  let highlight = $state({ x: 0, y: 0, width: 0, height: 0, slide: false, shown: false });
  let highlighted: Item[] | null = null;
  $effect(() => {
    void expanded; // an expanded row is taller
    const list = items;
    const row = document.getElementById(optionId(selected));
    const slide = list === highlighted;
    highlighted = list;
    if (!row) {
      highlight.shown = false;
      return;
    }
    highlight = { x: row.offsetLeft, y: row.offsetTop, width: row.offsetWidth, height: row.offsetHeight, slide, shown: true };
  });

  // The first rows of a list that has just appeared come in one after another; later answers replace them without it.
  function settling(node: HTMLElement) {
    node.classList.add("settling");
    const timer = setTimeout(() => node.classList.remove("settling"), 400);
    return { destroy: () => clearTimeout(timer) };
  }

  // The panel is only cloaked when closed, so the page keeps running: refocus and select the old
  // text every time the window regains the keyboard, so typing replaces it.
  function takeFocus() {
    active = true;
    shut = false;
    input.focus();
    if (performance.now() - filledAt < FILL_GRACE_MS) input.setSelectionRange(input.value.length, input.value.length);
    else input.select();
    refreshInfo();
    loadQuick(); // the apps may have been scanned, or the settings changed, since the bar last opened
  }

  // A hotkey that opens the bar with text typed in ("clip: ") tells the page just before the bar is shown. The
  // window gets several focus events while it opens, so for a second after the text arrives each of them leaves
  // it alone, with the cursor after it, instead of selecting it for the next keystroke to replace.
  const FILL_GRACE_MS = 1000;
  // Closed (not just blurred while opening): the bar's content waits hidden and rises in at the next opening.
  let shut = $state(false);
  let filledAt = -Infinity;
  $effect(() =>
    Events.On("panel:fill", (e) => {
      filledAt = performance.now();
      mode = "all";
      query = String(e.data ?? "");
      onInput();
      input?.focus();
      input?.setSelectionRange(query.length, query.length);
    }),
  );

  // The panel lost the keyboard, which is usually when it closes. But the window being activated sends a blur
  // of its own while the bar opens, and that must not wipe the text a hotkey typed into it ("clip: "), so
  // the bar counts as closed only if the keyboard has not come back a moment later.
  const CLOSE_CHECK_MS = 120;
  let closeTimer: ReturnType<typeof setTimeout> | undefined;
  function onBlur() {
    active = false;
    closeActions();
    picker = null;
    clearTimeout(closeTimer);
    closeTimer = setTimeout(() => !document.hasFocus() && closed(), CLOSE_CHECK_MS);
  }

  // The bar has closed: the next opening starts from everything, with the pinned things listed, and in the
  // next metal, chosen now, unseen.
  function closed() {
    quickPick = -1;
    shut = true;
    mode = "all";
    effective = "all";
    query = "";
    clearAnswer();
    loadPins();
    loadQuick();
    metal = nextMetal(metal, Math.random, rings.length);
  }

  // Glass: a soft light on the glass follows the pointer, as on a pane held up to a lamp. The card and the quick-open tiles are one
  // sheet of glass: the light is placed from the card's top left, and each tile shows the part of it that falls on the tile.
  let row: HTMLElement;
  let windowWidth = $state(window.innerWidth);
  function shine(event: PointerEvent) {
    if (theme !== "glass" || !card) return;
    const box = card.getBoundingClientRect();
    row.style.setProperty("--mx", `${event.clientX - box.left}px`);
    row.style.setProperty("--my", `${event.clientY - box.top}px`);
  }

  // Each quick-open tile shows the part of the pointer's light that falls on it (--ox, --oy: where the tile is from the card's top left),
  // with the card's glass behind it as a pane of its own.
  let card: HTMLElement;
  let tiles: HTMLElement[] = $state([]);
  $effect(() => {
    void cardWidth;
    void cardHeight;
    const frosted = theme === "glass";
    const count = quick.length;
    tick().then(() => {
      const shown = tiles.slice(0, count).filter(Boolean);
      if (!card || !frosted || shown.length === 0) {
        Panel.GlassPanes([], 0, 0);
        return;
      }
      const origin = card.getBoundingClientRect();
      for (const tile of shown) {
        const box = tile.getBoundingClientRect();
        tile.style.setProperty("--ox", `${box.left - origin.left}px`);
        tile.style.setProperty("--oy", `${box.top - origin.top}px`);
      }
      const first = shown[0].getBoundingClientRect();
      Panel.GlassPanes(
        shown.map((tile) => Math.round(tile.getBoundingClientRect().left)),
        Math.round(first.top),
        Math.round(first.width),
      );
    });
  });

  // The quick-open row: the apps set in Settings and the ones suggested for now (worked out on this computer only).
  async function loadQuick() {
    try {
      quick = (await Palette.QuickOpen()) ?? [];
    } catch {
      quick = [];
    }
  }

  // With nothing typed, Left and Right walk through the quick-open tiles (from the bar: Right to the first, Left to the last, and
  // past either end back to the bar), Return opens the one picked and Escape goes back to the bar. Once something is typed the
  // arrows move the caret again. True when the key was used.
  let quickPick = $state(-1);
  $effect(() => {
    if (query !== "" || quickPick >= quick.length) quickPick = -1;
  });
  function quickKey(event: KeyboardEvent): boolean {
    if (quick.length === 0 || query !== "" || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return false;
    const n = quick.length;
    switch (event.key) {
      case "ArrowRight":
        quickPick = quickPick === n - 1 ? -1 : quickPick + 1;
        return true;
      case "ArrowLeft":
        quickPick = quickPick === -1 ? n - 1 : quickPick - 1;
        return true;
      case "Enter":
        if (quickPick < 0) return false;
        openQuick(quickPick);
        quickPick = -1;
        return true;
      case "Escape":
        if (quickPick < 0) return false;
        quickPick = -1;
        return true;
    }
    return false;
  }

  // Opens the app in the slot (0-based) of the quick-open row; the launch is remembered like any other.
  function openQuick(slot: number) {
    const app = quick[slot];
    if (app) attempt(() => Palette.Do(app.ID, "open", ""));
  }

  // Cmd (Ctrl on Windows) with 1 to 3 (4) opens that quick-open slot; plain numbers are left to the search (a sum like 2+2). -1 for any other key.
  function quickSlot(event: KeyboardEvent): number {
    if (quick.length === 0 || event.altKey || event.shiftKey || !/^Digit[1-9]$/.test(event.code)) return -1;
    const slot = Number(event.code.slice(5)) - 1;
    if (slot >= quick.length) return -1;
    const command = isMac ? event.metaKey && !event.ctrlKey : event.ctrlKey && !event.metaKey;
    return command ? slot : -1;
  }

  // What is pinned, for the empty bar and for the marks on rows. Asked again whenever it may have changed.
  async function loadPins() {
    const [ids, rows] = await Promise.all([Palette.PinIDs(), Palette.Pinned()]);
    pins = new Set(ids ?? []);
    pinnedRows = rows ?? [];
    if (showingPins) {
      entries = pinnedRows;
      selected = Math.min(selected, Math.max(0, entries.length - 1));
    }
  }

  async function togglePin(item: Item | undefined) {
    const id = item && pinIdOf(item);
    if (!id) return;
    await attempt(async () => {
      await Palette.TogglePin(id);
      await loadPins();
    });
  }

  const pinningOf = (item: Item) => {
    const id = pinIdOf(item);
    return id === null ? null : pins.has(id) ? "unpin" : "pin";
  };

  const refreshInfo = () => Status.Info().then((i) => (info = i));

  let typingTimer: ReturnType<typeof setTimeout> | undefined;

  // The results fade in; in Glass they slide down as they fade in.
  function pour(_: Element) {
    const still = reduceMotion.matches;
    if (theme !== "glass") return { duration: still ? 0 : 160, delay: 60, css: (t: number) => `opacity: ${t}` };
    return {
      duration: still ? 0 : 280,
      delay: still ? 0 : 40,
      easing: cubicOut,
      css: (t: number, u: number) => `opacity: ${t}; transform: translateY(${u * -6}px)`,
    };
  }

  function onInput() {
    problem = "";
    typing = true; // the ring flows faster while you type, and settles a moment after you stop
    clearTimeout(typingTimer);
    typingTimer = setTimeout(() => (typing = false), 700);
    clearTimeout(timer);
    if (query.trim() === "") {
      clearAnswer();
      return;
    }
    update();
  }

  // Nothing to search for: drop the old answer at once, so the card goes back to what is pinned (or closes
  // into the capsule) and no half-answer ("nothing matched") flashes while the next word is typed.
  function clearAnswer() {
    token++;
    instant++;
    pending = false;
    results = null;
    answered = "";
    entries = mode === "all" ? pinnedRows : [];
    selected = 0;
    answerText = "";
    answerNote = "";
    effective = mode;
  }

  // Asks Go for the instant answers (apps), which come back in a millisecond, and then lets the files
  // follow after the usual pause. In the Apps, Web and Tags modes there is no file search.
  async function update() {
    const mine = ++instant;
    const answer = await Palette.Query(query, mode);
    if (mine !== instant) return; // a newer keystroke is already being answered
    entries = answer.Items ?? [];
    effective = answer.Mode as Mode;
    answerText = answer.Text;
    answerNote = answer.Note ?? "";
    selected = 0;
    clearTimeout(timer);
    if ((effective === "all" && !info?.FilesInBar && !asksForFiles(query)) || effective === "apps" || effective === "web" || effective === "tags" || effective === "clip" || effective === "snip" || effective === "note" || effective === "ask") {
      token++;
      pending = false;
      results = null;
      answered = "";
      return;
    }
    timer = setTimeout(runSearch, DEBOUNCE_MS);
  }

  function setMode(next: Mode) {
    mode = next;
    effective = next;
    if (query.trim() !== "") update();
    else clearAnswer();
  }

  async function runSearch() {
    const mine = ++token;
    const asked = query;
    pending = true;
    const answer = await Search.Search(asked, mine);
    if (answer.Stale || mine !== token) return; // a newer search is on its way
    pending = false;
    results = answer;
    answered = asked;
    selected = 0;
    expanded = new Set();
    if (answer.State === "warming" && active) {
      // The engine is starting (on demand, or just after launch): ask again until it answers, as typing more would.
      refreshInfo();
      timer = setTimeout(runSearch, WARMING_RETRY_MS);
    }
  }

  // The panel acts on the row that was selected when it opened, even if the pointer wanders over
  // others while it is open.
  function openActions() {
    if (current) target = current;
  }

  async function closeActions() {
    target = null;
    await tick();
    input?.focus();
  }

  async function runAction(id: string) {
    const item = target;
    await closeActions();
    if (!item) return;
    if (id === "copy-path") copyPath(item);
    else if (id === "tag") openPicker(item);
    else if (id === "pin") togglePin(item);
    else run(item, id as Way);
  }

  function copyPath(item: Item) {
    const path = filePathOf(item);
    if (!path) return;
    Clipboard.SetText(path);
    Panel.Hide(); // copied to be pasted elsewhere: give the keyboard back
  }

  // --- tags

  function openPicker(item: Item | undefined) {
    const id = item && tagIdOf(item);
    if (item && id) picker = { id, title: item.kind === "entry" ? item.entry.Title : item.row.Name };
  }

  async function closePicker() {
    picker = null;
    await tick();
    input?.focus();
  }

  function tagsChanged(id: string, tags: string[]) {
    tagsByID = { ...tagsByID, [id]: tags };
    if (mode === "tags" || query.trim().startsWith("#")) update(); // the list being looked at has changed
  }

  /** Runs something the user asked for; if it fails the reason is shown above the results. */
  async function attempt(action: () => Promise<unknown>, onFail?: () => void) {
    try {
      await action();
    } catch (error) {
      problem = String((error as Error)?.message ?? error);
      onFail?.();
    }
  }

  async function run(item: Item | undefined, how: Way = "open") {
    if (!item) return;
    if (how !== "open" && !available(item, how)) return;
    if (item.kind === "entry" && item.entry.Kind === "tag") {
      query = "#" + item.entry.ID.slice("tag:".length) + " ";
      onInput(); // show what carries it
      input.focus();
    } else if (item.kind === "entry" && item.entry.Fill) {
      query = item.entry.Fill; // a pinned quicklink or a built-in mode: the bar fills its trigger in and the rest is typed
      onInput();
      input.focus();
    } else if (item.kind === "entry") {
      const id = item.entry.ID;
      // An app that is gone has left the list, so the rows are asked for again.
      await attempt(() => Palette.Do(id, how, query), update);
      if (how === "delete") update(); // what was listed has changed
    } else if (how === "reveal") await attempt(() => Files.Reveal(pathOf(item)));
    else {
      const at = timeOf(item); // a moment in a recording opens in a player at that moment, when one can be asked to
      await attempt(() => (at === null ? Files.Open(pathOf(item), openLine(item)) : Files.OpenAt(pathOf(item), at)));
    }
  }

  /** Alt+Enter: search the web for what is typed, from any mode. */
  function searchTheWeb() {
    const text = (answerText || query).trim();
    if (text) attempt(() => Palette.Do("web:" + text, "open", query));
  }

  function toggle(open: boolean) {
    const row = current && current.kind !== "entry" ? current.row : undefined;
    if (!row || (row.Matches?.length ?? 0) < 2) return;
    const next = new Set(expanded);
    if (open) next.add(row.Path);
    else {
      next.delete(row.Path);
      if (current?.kind === "match") selected = items.findIndex((i) => i.kind === "row" && i.row === row);
    }
    expanded = next;
  }

  function atEndOfText() {
    return input.selectionStart === input.value.length && input.selectionEnd === input.value.length;
  }

  function onKeydown(event: KeyboardEvent) {
    if (target || picker) return; // a panel has the keyboard
    if (quickKey(event)) {
      event.preventDefault();
      return;
    }
    const slot = quickSlot(event);
    if (slot >= 0) {
      event.preventDefault();
      openQuick(slot);
      return;
    }
    const command = keyCommand(event, {
      current,
      atEndOfText: atEndOfText(),
      expanded: current !== undefined && current.kind !== "entry" && expanded.has(current.row.Path),
    });
    if (!command) return;
    event.preventDefault();
    switch (command.do) {
      case "settings":
        Panel.HideForLaunch(); // first: Hide() would hand the keyboard to the previous app and leave Settings behind it
        SettingsWindow.Open();
        break;
      case "escape": {
        const next = escapeStep(mode, query);
        if (next === "mode") setMode("all");
        else if (next === "text") {
          query = "";
          onInput();
        } else Panel.Hide();
        break;
      }
      case "mode":
        setMode(cycle(mode, command.by, info?.FileSearch));
        break;
      case "move":
        if (info?.SelectionEffect) {
          keying = true;
          clearTimeout(keyingTimer);
          keyingTimer = setTimeout(() => (keying = false), KEYING_MS);
        }
        selected = step(selected, command.by, items.length);
        scrollToSelected();
        break;
      case "web":
        searchTheWeb();
        break;
      case "run":
        run(current, command.how);
        break;
      case "expand":
      case "collapse":
        toggle(command.do === "expand");
        break;
      case "copy-path":
        if (current) copyPath(current);
        break;
      case "tag":
        openPicker(current);
        break;
      case "pin":
        togglePin(current);
        break;
      case "actions":
        openActions();
        break;
    }
  }

  // The list glides to the selected row: each frame covers a fixed share of what is left, so the motion is continuous at any
  // key-repeat rate. (CSS smooth scrolling restarts its animation on every repeat and moves in jerks while a key is held.)
  const GLIDE_MS = 70; // time constant: about 95% of the way in three of these
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
  let glideFrame = 0;
  let glideGoal = 0;

  async function scrollToSelected() {
    await tick();
    const box = document.getElementById("results");
    const row = box?.querySelector<HTMLElement>(".item.selected");
    if (!box || !row) return;
    // Where the list is heading counts as where it is, so a second key during a glide aims from the first one's goal.
    const at = glideFrame ? glideGoal : box.scrollTop;
    const shift = box.scrollTop - at;
    const edge = 8;
    const view = box.getBoundingClientRect();
    const rect = row.getBoundingClientRect();
    let goal = at;
    if (selected === 0) goal = 0; // the first row brings its heading with it
    else if (rect.top + shift < view.top + edge) goal = at + (rect.top + shift - view.top - edge);
    else if (rect.bottom + shift > view.bottom - edge) goal = at + (rect.bottom + shift - view.bottom + edge);
    goal = Math.min(Math.max(goal, 0), box.scrollHeight - box.clientHeight);
    if (reduceMotion.matches) {
      box.scrollTop = goal;
      return;
    }
    glideTo(box, goal);
  }

  function glideTo(box: HTMLElement, goal: number) {
    glideGoal = goal;
    if (glideFrame) return; // the running loop takes the new goal
    let pos = box.scrollTop;
    let last = performance.now();
    const frame = (now: number) => {
      // Something else moved the list (the wheel, new results): that wins, the glide ends.
      if (!box.isConnected || Math.abs(box.scrollTop - pos) > 2) {
        glideFrame = 0;
        return;
      }
      pos += (glideGoal - pos) * (1 - Math.exp(-Math.min(now - last, 50) / GLIDE_MS));
      last = now;
      if (Math.abs(glideGoal - pos) < 0.5) pos = glideGoal;
      box.scrollTop = pos;
      glideFrame = pos === glideGoal ? 0 : requestAnimationFrame(frame);
    };
    glideFrame = requestAnimationFrame(frame);
  }

  const optionId = (index: number) => `option-${index}`;

  onMount(() => {
    showUnexpectedIn((message) => (problem = message)); // inside the card, cleared by the next key like any failed action
    const stopTheme = onLook((look) => {
      if (look.Theme !== theme) (theme = look.Theme), (metal = 0);
      ring = look.Ring;
    });
    takeFocus();
    loadPins();
    loadQuick();
    // The observer fires after layout, whenever what the card holds changes size (results arriving,
    // fonts loading); the card and the window follow.
    const observer = new ResizeObserver(([entry]) => (contentHeight = entry.borderBoxSize[0].blockSize));
    observer.observe(content);
    // While something is loading the numbers move; keep the bar honest without waiting for a refocus.
    const poll = setInterval(() => active && waiting && refreshInfo(), 1500);
    return () => {
      stopTheme();
      observer.disconnect();
      clearInterval(poll);
    };
  });
</script>

<svelte:window bind:innerWidth={windowWidth} onfocus={takeFocus} onblur={onBlur} onkeydown={onKeydown} />

<main bind:this={row} style:--row-w="{Math.max(0, windowWidth - SIDE * 2)}px" style:--row-h="{cardHeight}px" role="presentation" onpointermove={shine}>
  <div class="card" class:ringed class:typing style:height="{cardHeight}px" use:glass={theme === "glass" ? 27 : 30} bind:this={card} bind:clientWidth={cardWidth}>
    {#if ringed}<LiquidRing speed={typing || pending ? 1.5 : 0.6} running={active} metal={rings[metal] ?? rings[0]} />{/if}
    <div class="surface">
      <div class="liquid" aria-hidden="true">
        <div class="flow">
          <span class="blob one"></span>
          <span class="blob two"></span>
          <span class="blob three"></span>
        </div>
      </div>
      <div class="content" class:acting={target || picker} class:shut bind:this={content}>
        <header class="bar">
          <span class="glass"><SearchGlyph size={22} busy={pending} /></span>
          <input
            bind:this={input}
            bind:value={query}
            oninput={onInput}
            type="text"
            placeholder={searchWhat}
            spellcheck="false"
            autocomplete="off"
            role="combobox"
            aria-label={searchWhat}
            aria-expanded={showPanel}
            aria-controls="results"
            aria-activedescendant={current ? optionId(selected) : undefined}
          />
          {#if waiting}
            <span class="status" transition:fade={{ duration: 160 }}><Loader text={waiting} /></span>
          {:else if chip}
            <span class="status scope">{chip}</span>
          {/if}
        </header>

        {#if showPanel}
          <div class="panel" in:pour>
            {#each notes as note}
              <p class="note" class:problem={note.problem}>{note.text}</p>
            {/each}
            {#if items.length > 0}
              <div class="results" class:stale={pending} class:keying id="results" role="listbox" aria-label="Results" use:settling>
                <div
                  class="highlight"
                  class:slide={highlight.slide}
                  hidden={!highlight.shown}
                  aria-hidden="true"
                  style:transform="translate({highlight.x}px, {highlight.y}px)"
                  style:width="{highlight.width}px"
                  style:height="{highlight.height}px"
                ></div>
                {#each items as item, index (keyOf(item))}
                  {#if showingPins && index === 0}<h3 class="section">Pinned</h3>{/if}
                  {#if sections && index === firstApp}<h3 class="section">Apps</h3>{/if}
                  {#if item.kind === "row" && item.heading}<h3 class="section">{item.heading}</h3>
                  {:else if sections && index === firstFile}<h3 class="section">Files</h3>{/if}
                  {#if item.kind === "entry"}
                    <EntryRow
                      id={optionId(index)}
                      entry={item.entry}
                      tags={tagsOf(item)}
                      pinned={pins.has(pinIdOf(item) ?? "")}
                      selected={index === selected}
                      onhover={() => ((selected = index), (keying = false))}
                      onactivate={() => run(item)}
                    />
                  {:else}
                    <ResultRow
                      id={optionId(index)}
                      row={item.row}
                      tags={tagsOf(item)}
                      pinned={pins.has(pinIdOf(item) ?? "")}
                      match={item.kind === "match" ? item.match : undefined}
                      query={answered}
                      selected={index === selected}
                      expanded={expanded.has(item.row.Path)}
                      onhover={() => ((selected = index), (keying = false))}
                      onactivate={() => run(item)}
                    />
                  {/if}
                {/each}
              </div>
              <Hints {current} {player} pinned={current && pinningOf(current) ? pinningOf(current) === "unpin" : null} {info} />
            {/if}
          </div>
        {/if}
      </div>
      {#if target}
        <ActionPanel actions={actionsFor(target, tagIdOf(target) !== null, pinningOf(target))} onrun={runAction} onclose={closeActions} />
      {:else if picker}
        <TagPicker id={picker.id} title={picker.title} {attempt} onchange={tagsChanged} onclose={closePicker} />
      {/if}
    </div>
  </div>
  {#if showQuick}
    <div class="quick" role="group" aria-label="Quick open" style:height="{quickSize}px" style:--tile-size="{quickSize}px" transition:fade={{ duration: 160 }}>
      {#each quick as app, index (app.ID)}
        <button type="button" class="tile" class:picked={quickPick === index} aria-current={quickPick === index} title="{app.Title}  {isMac ? "⌘" : "Ctrl+"}{index + 1}" aria-label="Open {app.Title} ({isMac ? "Command" : "Control"} {index + 1})" onclick={() => openQuick(index)} bind:this={tiles[index]} use:glassPane>
          <img src={app.Icon} alt="" draggable="false" />
        </button>
      {/each}
    </div>
  {/if}
  <p class="live" role="status" aria-live="polite">
    {#if results && items.length > 0}{items.length} results{/if}
  </p>
</main>

<style>
  main {
    box-sizing: border-box;
    display: flex;
    align-items: flex-start;
    gap: 8px;
    width: 100vw;
    padding: 12px 20px 40px;
  }
  .card {
    flex: 1 1 auto;
    min-width: 0;
    position: relative;
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    /* Height is animated on purpose: one small element, its content is absolutely positioned inside,
       and it only runs for 240 ms when results appear or go. */
    transition: height 240ms var(--ease);
  }
  .surface {
    position: absolute;
    inset: var(--ring);
    border-radius: calc(var(--radius) - var(--ring));
    background: var(--glass);
    isolation: isolate; /* the glow lies over the plate, under the bar */
  }
  .surface::after {
    content: "";
    position: absolute;
    inset: 0;
    border-radius: inherit;
    pointer-events: none;
    /* The edge: a lit top rim and a hairline all round, as on a machined card. */
    box-shadow:
      inset 0 1px 0 color-mix(in srgb, var(--ink) var(--bar-rim), transparent),
      inset 0 0 0 1px color-mix(in srgb, var(--ink) var(--bar-edge), transparent);
  }
  .content {
    position: relative;
    z-index: 1;
  }
  /* Opening: the card is there at once (its glass is the system's and cannot be animated), the bar's content rises into it.
     Fill only backwards: once it ends no opacity or transform is left on the text, which would draw it soft. */
  .content.shut {
    opacity: 0;
  }
  .content:not(.shut) {
    animation: rise 200ms var(--ease) backwards;
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(5px);
    }
  }
  /* Room for the action panel above the footer, however few rows there are. */
  .content.acting {
    min-height: 270px;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 14px;
    box-sizing: border-box;
    height: 60px;
    /* The display font sits its letters high in the line, so the content is eased down to look centred. */
    padding: 3px 24px 0 24px;
    /* The bar is moved by holding it anywhere but the text field; the panel remembers where it was left. */
    --wails-draggable: drag;
  }
  .glass {
    flex: none;
    cursor: grab;
    display: grid;
    color: var(--text-3);
    transition: color 200ms var(--ease);
  }
  .bar:focus-within .glass {
    color: var(--chrome);
  }
  :global(html[data-theme="glass"]) .bar:focus-within .glass {
    color: var(--chrome);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    outline: 0;
    background: transparent;
    --wails-draggable: no-drag;
    color: var(--text);
    font: 400 19px/1 var(--display);
    letter-spacing: -0.01em;
  }
  input::placeholder {
    color: var(--text-3);
  }
  .status {
    flex: none;
    margin-left: auto;
  }
  .scope {
    padding: 3px 11px;
    border-radius: 999px;
    background: var(--tile);
    color: var(--text-2);
    font: 500 11.5px var(--mono);
  }
  .panel {
    border-top: 1px solid var(--line);
  }
  .note {
    margin: 0;
    padding: 12px 26px 4px;
    color: var(--text-2);
    font-size: 12.5px;
  }
  .note.problem {
    color: var(--danger);
  }
  .results {
    max-height: 440px;
    overflow-y: auto;
    padding: 8px 0;
    animation: unfold 240ms var(--ease) backwards; /* when the list appears, not on every keystroke */
  }
  @keyframes unfold {
    from {
      opacity: 0;
      transform: translateY(-6px);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .content:not(.shut),
    .results {
      animation: none;
    }
  }
  /* Rows are faded by the colour of their text, never by opacity or a transform: those make the browser draw the row as a
     picture that is then scaled or blended, which is what made it look soft. Dimmed text is the same glyphs in a fainter
     colour, so it stays as sharp as ever. */
  .results {
    --dim-text: color-mix(in srgb, var(--text) 45%, transparent);
    --dim-text-3: color-mix(in srgb, var(--text-3) 55%, transparent);
  }
  .results :global(.item) {
    transition:
      background-color 200ms var(--ease),
      box-shadow 240ms var(--ease);
  }
  .results :global(.item .name),
  .results :global(.item .kind),
  .results :global(.item .folder),
  .results :global(.item .detail),
  .results :global(.item .where),
  .results :global(.item mark),
  .results :global(.item .tag),
  .results :global(.item .pin),
  .results :global(.item .tile) {
    transition:
      color 200ms var(--ease),
      opacity 200ms var(--ease);
  }
  /* While a search is in flight the list shows the answer to an older query, a little fainter. While the arrow keys move the
     selection (and for a moment after) the selected row is a lit pane of glass and the rest fade back. */
  .results.stale :global(.item .name),
  .results.stale :global(.item mark),
  .results.keying :global(.item:not(.selected) .name),
  .results.keying :global(.item:not(.selected) mark) {
    color: var(--dim-text);
  }
  .results.stale :global(.item .kind),
  .results.stale :global(.item .folder),
  .results.stale :global(.item .detail),
  .results.stale :global(.item .where),
  .results.stale :global(.item .tag),
  .results.stale :global(.item .pin),
  .results.keying :global(.item:not(.selected) .kind),
  .results.keying :global(.item:not(.selected) .folder),
  .results.keying :global(.item:not(.selected) .detail),
  .results.keying :global(.item:not(.selected) .where),
  .results.keying :global(.item:not(.selected) .tag),
  .results.keying :global(.item:not(.selected) .pin) {
    color: var(--dim-text-3);
  }
  .results.keying :global(.item:not(.selected) .tile) {
    opacity: 0.5; /* the tile holds no text: only an icon */
  }
  /* The rows lie over the one highlight (it carries what a selected row used to paint itself). */
  .results {
    position: relative;
  }
  .results :global(.item) {
    position: relative;
  }
  .results :global(.item.selected) {
    background: none;
    box-shadow: none;
  }
  .highlight {
    position: absolute;
    top: 0;
    left: 0;
    border-radius: 16px;
    background: var(--lift);
    box-shadow: inset 0 1px 0 color-mix(in srgb, var(--ink) 5%, transparent);
    pointer-events: none;
    transition:
      background-color 200ms var(--ease),
      box-shadow 240ms var(--ease);
  }
  .highlight.slide {
    transition:
      transform 220ms var(--spring),
      height 220ms var(--spring),
      background-color 200ms var(--ease),
      box-shadow 240ms var(--ease);
  }
  .results.keying .highlight {
    background-color: var(--lift-keyed);
    box-shadow:
      inset 0 1px 0 color-mix(in srgb, var(--ink) 20%, transparent),
      0 0 0 1px color-mix(in srgb, var(--ink) 8%, transparent),
      0 8px 16px -8px rgba(0, 0, 0, 0.95);
  }
  /* Pressing a row: the highlight darkens and the icon sinks, then springs back on release. Only the tile moves (it holds no
     text, so nothing is drawn soft). */
  .results:active .highlight {
    background-color: var(--lift-keyed);
    transition-duration: 60ms;
  }
  .results :global(.item .tile) {
    transition: transform 280ms var(--spring);
  }
  .results :global(.item:active .tile) {
    transform: scale(0.9);
    transition-duration: 80ms;
  }
  /* A list that has just appeared: its first rows come in 25 ms apart (only then, never on a keystroke). */
  .results:global(.settling) > :global(*:not(.highlight)) {
    animation: rowin 240ms var(--ease) backwards;
  }
  .results:global(.settling) > :global(:nth-child(3)) {
    animation-delay: 25ms;
  }
  .results:global(.settling) > :global(:nth-child(4)) {
    animation-delay: 50ms;
  }
  .results:global(.settling) > :global(:nth-child(5)) {
    animation-delay: 75ms;
  }
  .results:global(.settling) > :global(:nth-child(n + 6)) {
    animation-delay: 100ms;
  }
  @keyframes rowin {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .highlight,
    .highlight.slide,
    .results :global(.item .tile) {
      transition: none;
    }
    .results:global(.settling) > :global(*) {
      animation: none;
    }
  }
  .section {
    margin: 4px 24px 2px;
    color: var(--text-3);
    font: 500 10.5px var(--mono);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .section:not(.highlight + .section) {
    margin-top: 10px;
  }
  .results::-webkit-scrollbar {
    width: 10px;
  }
  .results::-webkit-scrollbar-thumb {
    border: 3px solid transparent;
    border-radius: 999px;
    background: color-mix(in srgb, var(--ink) 14%, transparent) padding-box;
  }
  /* Glass: no metal and no bevel, and a slimmer bar. The plate reaches the card's edge, which is a hairline in the accent (or nothing,
     if the bar is set to have no edge), and the light under the text moves like a slow liquid: two faint blobs of the accent wander
     over the whole card, results and all, and change shape as they go, each pass taking about a quarter of a minute, brighter while
     the bar has the keyboard and while you type. Nothing in it moves fast enough to pull the eye from what is being typed. */
  .liquid {
    display: none;
  }
  :global(html[data-theme="glass"]) .card {
    box-shadow: 0 18px 40px -14px rgba(0, 0, 0, 0.5);
  }
  :global(html[data-theme="glass"]) .surface {
    inset: 0;
    border-radius: var(--radius);
  }
  :global(html[data-theme="glass"]) .content {
    top: var(--ring); /* the card is still as tall as a ringed one: keep the bar in its middle */
  }
  :global(html[data-theme="glass"]) .bar {
    height: 50px;
    gap: 12px;
    padding: 0 20px;
  }
  :global(html[data-theme="glass"]) input {
    caret-color: var(--chrome);
    font-size: 16.5px;
    font-weight: 400;
  }
  :global(html[data-theme="glass"]) .surface::after {
    box-shadow: none;
    transition: box-shadow 900ms var(--ease);
  }
  /* Every state of the edge asks for it to be on, so hovering or typing never brings back an edge that was turned off. */
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .surface::after {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 22%, transparent);
  }
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .card:hover .surface::after {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 25%, transparent);
  }
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .card:focus-within .surface::after {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 38%, transparent);
  }
  :global(html[data-theme="glass"]) .liquid {
    display: block;
    position: absolute;
    inset: 0;
    overflow: hidden;
    border-radius: inherit;
    pointer-events: none;
    opacity: 0.22;
    transition: opacity 1200ms var(--ease);
  }
  /* The flow is the whole row, card and quick-open tiles, placed from the card's top left; the card and each tile show their own part
     of it. The blobs travel in shares of it (cqw, cqh), so they cross the card however tall the results make it and go on through the
     tiles. */
  .flow {
    position: absolute;
    left: calc(-1 * var(--ox, 0px));
    top: calc(-1 * var(--oy, 0px));
    width: var(--row-w, 100%);
    height: var(--row-h, 100%);
    container-type: size;
  }
  :global(html[data-theme="glass"]) main:has(.card:hover, .tile:hover) .liquid {
    opacity: 0.27;
  }
  :global(html[data-theme="glass"]) main:has(.card:focus-within) .liquid {
    opacity: 0.42;
  }
  :global(html[data-theme="glass"]) main:has(.card.typing) .liquid {
    opacity: 0.55;
    transition-duration: 500ms;
  }
  .blob {
    position: absolute;
    top: -70px; /* the blob's middle is the card's top left corner, from where the paths below move it */
    left: -110px;
    width: 220px;
    height: 140px;
    background: radial-gradient(closest-side, var(--chrome), transparent);
    animation: wander 11s ease-in-out infinite alternate;
  }
  .blob.two {
    width: 280px;
    left: -140px;
    animation-name: wander-back;
    animation-duration: 15s;
    animation-delay: -6s;
  }
  .blob.three {
    width: 160px;
    height: 110px;
    left: -80px;
    top: -55px;
    background: radial-gradient(closest-side, color-mix(in srgb, var(--chrome) 55%, white), transparent);
    animation-duration: 8s;
    animation-delay: -3s;
  }
  @media (prefers-reduced-motion: reduce) {
    .blob {
      animation: none;
    }
  }
  @keyframes wander {
    0% {
      transform: translate(8cqw, 25cqh) scale(1);
    }
    33% {
      transform: translate(45cqw, 80cqh) scale(1.28, 0.86);
    }
    66% {
      transform: translate(72cqw, 20cqh) scale(1.1, 1.05);
    }
    100% {
      transform: translate(95cqw, 65cqh) scale(0.86, 1.16);
    }
  }
  @keyframes wander-back {
    0% {
      transform: translate(90cqw, 75cqh) scale(1.1, 0.9);
    }
    33% {
      transform: translate(55cqw, 15cqh) scale(0.9, 1.15);
    }
    66% {
      transform: translate(25cqw, 70cqh) scale(1.25, 0.9);
    }
    100% {
      transform: translate(6cqw, 30cqh) scale(1, 1.1);
    }
  }
  /* Liquid Glass: a lit rim that is brightest where light falls on a curved edge (top left, bottom right), a little depth
     inside the pane, a highlight that follows the pointer, and the quick-open tiles as small panes of their own. */
  :global(html[data-theme="glass"]) .surface {
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.24),
      inset 0 -1px 0 rgba(0, 0, 0, 0.2),
      inset 0 0 28px rgba(255, 255, 255, 0.05);
  }
  :global(html[data-theme="glass"]) .surface::before {
    content: "";
    position: absolute;
    inset: 0;
    z-index: 2;
    padding: 1px;
    border-radius: inherit;
    pointer-events: none;
    background: linear-gradient(
      155deg,
      rgba(255, 255, 255, 0.62),
      rgba(255, 255, 255, 0.12) 26%,
      rgba(255, 255, 255, 0) 50%,
      rgba(255, 255, 255, 0.08) 76%,
      rgba(255, 255, 255, 0.38)
    );
    -webkit-mask:
      linear-gradient(#000 0 0) content-box,
      linear-gradient(#000 0 0);
    mask:
      linear-gradient(#000 0 0) content-box,
      linear-gradient(#000 0 0);
    -webkit-mask-composite: xor;
    mask-composite: exclude;
    opacity: 0.75;
  }
  :global(html[data-theme="glass"]) .card::after {
    content: "";
    position: absolute;
    inset: 0;
    z-index: 2;
    border-radius: var(--radius);
    pointer-events: none;
    background: radial-gradient(360px circle at var(--mx, 30%) var(--my, 0%), rgba(255, 255, 255, 0.05), transparent 70%);
    opacity: 0;
    transition: opacity 1200ms var(--ease);
  }
  :global(html[data-theme="glass"]) main:has(.card:hover, .tile:hover) .card::after {
    opacity: 1;
  }
  /* The quick-open tiles are made of the card's glass: the same blur behind (a pane of its own, from the launcher), plate, lit rim,
     accent hairline, liquid and pointer light. They do not grow on hover, which would pull them off the blur behind them. */
  :global(html[data-theme="glass"]) .tile {
    position: relative;
    isolation: isolate;
    border-radius: 50%;
    background: var(--glass);
    box-shadow:
      0 18px 40px -14px rgba(0, 0, 0, 0.5),
      inset 0 1px 0 rgba(255, 255, 255, 0.24),
      inset 0 -1px 0 rgba(0, 0, 0, 0.2),
      inset 0 0 28px rgba(255, 255, 255, 0.05);
  }
  :global(html[data-theme="glass"]) .tile:hover,
  :global(html[data-theme="glass"]) .tile.picked,
  :global(html[data-theme="glass"]) .tile:active {
    transform: none;
  }
  :global(html[data-theme="glass"]) .tile::before {
    content: "";
    position: absolute;
    inset: 0;
    z-index: 2;
    padding: 1px;
    border-radius: inherit;
    pointer-events: none;
    background: linear-gradient(
      155deg,
      rgba(255, 255, 255, 0.62),
      rgba(255, 255, 255, 0.12) 26%,
      rgba(255, 255, 255, 0) 50%,
      rgba(255, 255, 255, 0.08) 76%,
      rgba(255, 255, 255, 0.38)
    );
    -webkit-mask:
      linear-gradient(#000 0 0) content-box,
      linear-gradient(#000 0 0);
    mask:
      linear-gradient(#000 0 0) content-box,
      linear-gradient(#000 0 0);
    -webkit-mask-composite: xor;
    mask-composite: exclude;
    opacity: 0.75;
  }
  /* The pointer light (from the card's top left, moved to the tile's) and the accent hairline, which brightens under the pointer. */
  :global(html[data-theme="glass"]) .tile::after {
    content: "";
    position: absolute;
    inset: 0;
    z-index: 2;
    border-radius: inherit;
    pointer-events: none;
    background: radial-gradient(
      360px circle at calc(var(--mx, -999px) - var(--ox, 0px)) calc(var(--my, -999px) - var(--oy, 0px)),
      rgba(255, 255, 255, 0.05),
      transparent 70%
    );
    transition:
      box-shadow 900ms var(--ease),
      background-color 360ms var(--ease);
  }
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .tile::after {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 22%, transparent);
  }
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .tile:hover::after,
  :global(html[data-theme="glass"]:not([data-bar-border="false"])) .tile.picked::after {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 38%, transparent);
  }
  :global(html[data-theme="glass"]) .tile:hover::after {
    background-color: rgba(255, 255, 255, 0.05);
  }
  /* Picked with the arrow keys: lit like a hovered tile, a little more, and the icon lifts. */
  :global(html[data-theme="glass"]) .tile.picked::after {
    background-color: rgba(255, 255, 255, 0.09);
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, var(--chrome) 55%, transparent),
      0 0 18px -2px color-mix(in srgb, var(--chrome) 45%, transparent);
  }
  :global(html[data-theme="glass"]) .tile.picked img {
    transform: translateY(-1px) scale(1.06);
  }
  :global(html[data-theme="glass"]) .tile:active::after {
    background-color: rgba(255, 255, 255, 0.1);
    transition-duration: 0ms;
  }
  :global(html[data-theme="glass"]) .tile img {
    position: relative;
    z-index: 1;
    transition: transform 220ms var(--ease);
  }
  :global(html[data-theme="glass"]) .tile:active img {
    transform: scale(0.92);
  }
  @media (prefers-reduced-transparency: reduce) {
    :global(html[data-theme="glass"]) .card::after,
    :global(html[data-theme="glass"]) .surface::before,
    :global(html[data-theme="glass"]) .tile::before {
      display: none;
    }
  }
  /* Rows hovered in Glass light up through the same liquid easing. */
  :global(html[data-theme="glass"]) .results :global(.item) {
    transition:
      background-color 360ms var(--ease),
      box-shadow 360ms var(--ease);
  }
  /* Quick open: a small rounded tile per app beside the bar, the icon and nothing else (the name is its tooltip). */
  .quick {
    display: flex;
    flex: none;
    align-items: center;
    gap: 8px;
  }
  .tile {
    display: grid;
    place-items: center;
    width: var(--tile-size, 50px); /* as tall as the bar beside it */
    height: var(--tile-size, 50px);
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: var(--plate);
    box-shadow:
      0 6px 14px -6px rgba(0, 0, 0, 0.55),
      inset 0 0 0 1px var(--line);
    cursor: pointer;
    --wails-draggable: no-drag;
    transition:
      transform 220ms var(--ease),
      box-shadow 220ms var(--ease);
  }
  .tile:hover,
  .tile.picked {
    transform: scale(1.04);
  }
  .tile:active {
    transform: scale(0.96);
  }
  :global(html:not([data-theme="glass"])) .tile.picked {
    box-shadow:
      0 6px 14px -6px rgba(0, 0, 0, 0.55),
      inset 0 0 0 1.5px var(--chrome);
  }
  .tile:focus-visible {
    outline: 1px solid var(--chrome);
    outline-offset: 2px;
  }
  .tile img {
    width: calc(var(--tile-size, 50px) * 0.52); /* the circle is a plate round the icon, not a frame: room to breathe */
    height: calc(var(--tile-size, 50px) * 0.52);
    pointer-events: none;
  }
  .live {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
</style>
