<script lang="ts">
  import { isMac, words } from "../platform";
  import { onMount } from "svelte";
  import { OmniAI, Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { ClipboardView, NotesHotkeyView, OmniAIView, View } from "../../../bindings/omni/internal/settingsui";
  import HotkeyField from "./HotkeyField.svelte";
  import { ACCENTS, applyLook, THEMES } from "../theme";

  let { view }: { view: View } = $props();

  // The form starts from the settings as they were when the page opened and then owns its own values.
  // svelte-ignore state_referenced_locally
  let hotkey = $state(view.Hotkey);
  // svelte-ignore state_referenced_locally
  let browser = $state(view.Browser);
  // svelte-ignore state_referenced_locally
  let searchURL = $state(view.SearchURL);
  // svelte-ignore state_referenced_locally
  let startup = $state(view.Autostart);
  // svelte-ignore state_referenced_locally
  let lift = $state(view.SelectionEffect);
  // svelte-ignore state_referenced_locally
  let theme = $state(view.Theme);
  // svelte-ignore state_referenced_locally
  let ring = $state(view.Ring);
  // svelte-ignore state_referenced_locally
  let lights = $state(view.Lights);
  // svelte-ignore state_referenced_locally
  let accent = $state(view.Accent);
  let browsers = $state<string[]>([]);
  // The other hotkeys live on pages of their own services; they are shown here, beside the search bar's, and saved there.
  let clipboard = $state<ClipboardView | null>(null);
  let chat = $state<OmniAIView | null>(null);
  let notes = $state<NotesHotkeyView | null>(null);
  let error = $state("");
  let saved = $state(false);
  let savedTimer: ReturnType<typeof setTimeout> | undefined;

  const NAMES: Record<string, string> = {
    chrome: "Chrome",
    msedge: "Edge",
    firefox: "Firefox",
    zen: "Zen",
    brave: "Brave",
    vivaldi: "Vivaldi",
    opera: "Opera",
    librewolf: "LibreWolf",
    waterfox: "Waterfox",
    floorp: "Floorp",
  };
  // A browser set by path in config.toml is not in the installed list; keep it selectable.
  const options = $derived(browser === "" || browsers.includes(browser) ? browsers : [browser, ...browsers]);
  const QUERY = "{q}";

  async function loadHotkeys() {
    [clipboard, chat, notes] = await Promise.all([Settings.Clipboard(), OmniAI.View(), Settings.NotesHotkey()]);
  }

  onMount(async () => {
    loadHotkeys();
    browsers = (await Settings.Browsers()) ?? [];
  });

  function flashSaved() {
    saved = true;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => (saved = false), 1800);
  }

  async function save() {
    error = await Settings.SaveGeneral(hotkey, browser, searchURL);
    if (error === "") flashSaved();
  }

  // The hotkey field shows its own refusal ("that hotkey is taken"); the page only learns what was saved.
  async function saveHotkey(accelerator: string): Promise<string> {
    const refused = await Settings.SaveGeneral(accelerator, browser, searchURL);
    if (refused === "") {
      hotkey = accelerator;
      flashSaved();
    }
    return refused;
  }

  // Each field shows its own refusal; the page reloads the hotkeys so an error from startup goes away once the key works.
  async function afterKey(refused: string): Promise<string> {
    if (refused === "") {
      await loadHotkeys();
      flashSaved();
    }
    return refused;
  }
  const saveClipboardKey = async (accelerator: string) => {
    const now = await Settings.Clipboard();
    return afterKey(await Settings.SaveClipboard(now.Enabled, now.Ignore ?? [], accelerator));
  };
  const saveChatKey = async (accelerator: string) => afterKey(await OmniAI.SetHotkey(accelerator));
  const saveNotesKey = async (accelerator: string) => afterKey(await Settings.SetNotesHotkey(accelerator));

  async function saveLift() {
    error = await Settings.SetSelectionEffect(lift);
    if (error !== "") lift = !lift;
    else flashSaved();
  }

  async function saveTheme(id: string) {
    const before = theme;
    theme = id;
    applyLook({ Theme: id });
    error = await Settings.SetTheme(id);
    if (error !== "") {
      theme = before;
      applyLook({ Theme: before });
    } else flashSaved();
  }

  async function saveLights() {
    applyLook({ Lights: lights });
    error = await Settings.SetLights(lights);
    if (error !== "") applyLook({ Lights: (lights = !lights) });
  }

  async function saveRing() {
    applyLook({ Ring: ring });
    error = await Settings.SetRing(ring);
    if (error !== "") applyLook({ Ring: (ring = !ring) });
    else flashSaved();
  }

  async function saveAccent(id: string) {
    const before = accent;
    accent = id;
    applyLook({ Accent: id });
    error = await Settings.SetAccent(id);
    if (error !== "") {
      accent = before;
      applyLook({ Accent: before });
    } else flashSaved();
  }

  async function saveStartup() {
    error = await Settings.SetAutostart(startup);
    if (error !== "") startup = !startup;
    else flashSaved();
  }
</script>

<div class="head-row">
  <h2>General</h2>
  <p class="ok" aria-live="polite">{saved ? "Saved" : ""}</p>
</div>

<h3 class="group-title">Appearance</h3>
<div class="group">
  <div class="item stack">
    <span class="name" id="theme-label">Theme</span>
    <div class="themes" role="radiogroup" aria-labelledby="theme-label">
      {#each THEMES as t (t.id)}
        <button type="button" role="radio" aria-checked={theme === t.id} class:on={theme === t.id} onclick={() => saveTheme(t.id)}>
          <span class="swatch" style:background={t.plate} style:color={t.text}>Aa</span>
          {t.label}
        </button>
      {/each}
    </div>
    {#if theme === "glass" && !isMac}
      <span class="sub">
        Windows pauses its blur while Energy saver is on: the cards show your wallpaper, blurred, instead of what is behind them
        until it is off. With Transparency effects off (Personalization, Colors) they are solid.
      </span>
    {/if}
  </div>
  <div class="item stack">
    <span class="name" id="accent-label">Accent</span>
    <div class="themes" role="radiogroup" aria-labelledby="accent-label">
      {#each ACCENTS as a (a.id)}
        <button type="button" role="radio" aria-checked={accent === a.id} class:on={accent === a.id} onclick={() => saveAccent(a.id)}>
          <span class="dot" style:background={a.swatch}></span>
          {a.label}
        </button>
      {/each}
    </div>
  </div>
  <label class="item">
    <span class="text">
      <span class="name">Metal ring</span>
      <span class="sub">The flowing ring around the search bar and the Omni AI box</span>
    </span>
    <input type="checkbox" class="switch" bind:checked={ring} onchange={saveRing} />
  </label>
  {#if theme === "glass"}
    <label class="item">
      <span class="text">
        <span class="name">Floating lights</span>
        <span class="sub">Two faint lights in the accent's colour drifting inside the search bar</span>
      </span>
      <input type="checkbox" class="switch" bind:checked={lights} onchange={saveLights} />
    </label>
  {/if}
  <label class="item">
    <span class="text">
      <span class="name">Lift the selected result</span>
      <span class="sub">While moving through results with the arrow keys</span>
    </span>
    <input type="checkbox" class="switch" bind:checked={lift} onchange={saveLift} />
  </label>
</div>

<h3 class="group-title">Hotkeys</h3>
<div class="group">
  <div class="item">
    <span class="name" id="hotkey-label">Search bar</span>
    <HotkeyField labelId="hotkey-label" value={view.Hotkey} defaultValue={view.DefaultHotkey} onsave={saveHotkey} />
  </div>
  {#if clipboard && chat && notes}
    <div class="item">
      <span class="name" id="clip-hotkey-label">Clipboard history</span>
      <HotkeyField labelId="clip-hotkey-label" value={clipboard.Hotkey} clearable onsave={saveClipboardKey} />
    </div>
    {#if clipboard.HotkeyError}<p class="item error">{clipboard.HotkeyError}</p>{/if}
    <div class="item">
      <span class="name" id="ai-hotkey-label">Omni AI</span>
      <HotkeyField labelId="ai-hotkey-label" value={chat.Hotkey} clearable onsave={saveChatKey} />
    </div>
    {#if chat.HotkeyError}<p class="item error">{chat.HotkeyError}</p>{/if}
    <div class="item">
      <span class="name" id="notes-hotkey-label">Notes</span>
      <HotkeyField labelId="notes-hotkey-label" value={notes.Hotkey} clearable onsave={saveNotesKey} />
    </div>
    {#if notes.Error}<p class="item error">{notes.Error}</p>{/if}
  {/if}
</div>
{#if !isMac}<p class="group-note">Windows-key combinations go in config.toml.</p>{/if}

<h3 class="group-title">Web</h3>
<div class="group">
  <div class="item">
    <label class="name" for="browser">Browser</label>
    <select id="browser" bind:value={browser} onchange={() => save()}>
      <option value="">{words().defaultBrowser}</option>
      {#each options as name (name)}
        <option value={name}>{NAMES[name] ?? name}</option>
      {/each}
    </select>
  </div>
  <div class="item">
    <span class="text">
      <label class="name" for="search-url">Search address</label>
      <span class="sub">{QUERY} marks the words. Empty uses the browser's own engine, or DuckDuckGo.</span>
    </span>
    <input id="search-url" type="text" class="mono" placeholder={"https://duckduckgo.com/?q=" + QUERY} bind:value={searchURL} onchange={() => save()} spellcheck="false" />
  </div>
</div>

<h3 class="group-title">System</h3>
<div class="group">
  <label class="item">
    <span class="name">Start when I sign in</span>
    <input type="checkbox" class="switch" bind:checked={startup} onchange={saveStartup} />
  </label>
</div>

{#if error}<p class="error">{error}</p>{/if}

<style>
  .item.error {
    min-height: 0;
    padding-top: 0;
    border-top: 0;
    font-size: 12px;
  }
  .themes {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .themes button {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 12px 5px 5px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--tile);
    color: var(--text-2);
    font: 500 13px/1 var(--sans);
    cursor: pointer;
  }
  .themes button:hover {
    background: var(--lift);
  }
  .themes button.on {
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
    color: var(--text);
  }
  .themes button:focus-visible {
    outline: 1px solid var(--chrome);
    outline-offset: 2px;
  }
  .dot {
    width: 14px;
    height: 14px;
    margin: 6px 0 6px 6px;
    border-radius: 50%;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink) 18%, transparent);
  }
  .swatch {
    display: grid;
    place-items: center;
    width: 30px;
    height: 26px;
    border: 1px solid color-mix(in srgb, var(--ink) 25%, transparent);
    border-radius: 7px;
    font: 620 12px/1 var(--sans);
  }
  label.item {
    cursor: pointer;
  }
</style>
