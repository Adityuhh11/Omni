<script lang="ts">
  import { isMac, words } from "../platform";
  import { onMount } from "svelte";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { AppChoice, View } from "../../../bindings/omni/internal/settingsui";
  import type { QuickOpen } from "../../../bindings/omni/internal/config";
  import { ACCENTS, applyLook, THEMES } from "../theme";

  let { view }: { view: View } = $props();

  // The form starts from the settings as they were when the page opened and then owns its own values.
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
  let accent = $state(view.Accent);
  let barBorder = $state(true);
  let quick = $state<QuickOpen | null>(null);
  let appChoices = $state<AppChoice[]>([]);
  let forgotten = $state(false);
  let browsers = $state<string[]>([]);
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

  onMount(async () => {
    Settings.Look().then((look) => (barBorder = look.BarBorder));
    Settings.QuickOpen().then((q) => (quick = q));
    Settings.QuickOpenApps().then((list) => (appChoices = list ?? []));
    browsers = (await Settings.Browsers()) ?? [];
  });

  // The slot's app as the picker shows it: "" is "Suggested".
  const slotApp = (q: QuickOpen, slot: number) => q.Apps?.[slot] ?? "";

  async function saveQuick(change: Partial<QuickOpen>) {
    if (!quick) return;
    const before = quick;
    const next: QuickOpen = { ...quick, ...change };
    quick = next;
    error = await Settings.SaveQuickOpen(next);
    if (error !== "") quick = before;
    else flashSaved();
  }

  function setSlot(slot: number, id: string) {
    if (!quick) return;
    const apps = Array.from({ length: Math.max(quick.Apps?.length ?? 0, slot + 1) }, (_, i) => slotApp(quick!, i));
    apps[slot] = id;
    saveQuick({ Apps: apps });
  }

  async function forgetUsage() {
    error = await Settings.ForgetUsage();
    if (error === "") {
      forgotten = true;
      setTimeout(() => (forgotten = false), 2400);
    }
  }

  function flashSaved() {
    saved = true;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => (saved = false), 1800);
  }

  // The bar's hotkey is saved with these two but lives on the Hotkeys page, which may have changed it since this window opened.
  async function save() {
    error = await Settings.SaveGeneral((await Settings.View()).Hotkey, browser, searchURL);
    if (error === "") flashSaved();
  }

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

  // A colour of their own is shown on the page while the picker is still open, and saved when it lets go.
  const previewAccent = (hex: string) => applyLook({ Accent: hex });
  const isCustom = $derived(!ACCENTS.some((a) => a.id === accent));

  async function saveBarBorder() {
    applyLook({ BarBorder: barBorder });
    error = await Settings.SetBarBorder(barBorder);
    if (error !== "") applyLook({ BarBorder: (barBorder = !barBorder) });
    else flashSaved();
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
        Glass needs Windows' blur. While Energy saver (battery saver) is on, Windows turns the blur off and the cards go almost solid until
        it is off again. With Transparency effects off (Personalization, Colors) they are solid.
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
      <label class="custom" class:on={isCustom} title="Any colour">
        <span class="dot" style:background={isCustom ? accent : "conic-gradient(#e5484d, #f5d90a, #46a758, #5bb8d0, #3e63dd, #8e4ec6, #d6409f, #e5484d)"}></span>
        Custom
        <input
          type="color"
          value={isCustom ? accent : "#f2601d"}
          aria-label="Any colour"
          oninput={(e) => previewAccent(e.currentTarget.value)}
          onchange={(e) => saveAccent(e.currentTarget.value)}
        />
      </label>
    </div>
  </div>
  {#if theme !== "glass"}
    <label class="item">
      <span class="text">
        <span class="name">Metal ring</span>
        <span class="sub">The flowing ring around the search bar and the Omni AI box</span>
      </span>
      <input type="checkbox" class="switch" bind:checked={ring} onchange={saveRing} />
    </label>
  {/if}
  {#if theme === "glass"}
    <label class="item">
      <span class="text">
        <span class="name">Bar edge</span>
        <span class="sub">A thin line in the accent colour round the search bar; off, the bar only glows while you type</span>
      </span>
      <input type="checkbox" class="switch" bind:checked={barBorder} onchange={saveBarBorder} />
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

{#if quick}
  <h3 class="group-title">Quick open</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Show app shortcuts beside the bar</span>
        <span class="sub">{isMac ? "⌘1" : "Ctrl+1"} to {isMac ? "⌘" : "Ctrl+"}{quick.Count} opens them, typing or not</span>
      </span>
      <input type="checkbox" class="switch" checked={quick.Enabled} onchange={(e) => saveQuick({ Enabled: e.currentTarget.checked })} />
    </label>
    {#if quick.Enabled}
      <div class="item">
        <label class="name" for="quick-count">Number of apps</label>
        <select id="quick-count" value={quick.Count} onchange={(e) => saveQuick({ Count: Number(e.currentTarget.value) })}>
          <option value={3}>3</option>
          <option value={4}>4</option>
        </select>
      </div>
      {#each Array.from({ length: quick.Count }, (_, i) => i) as slot (slot)}
        <div class="item">
          <label class="name" for="quick-slot-{slot}">Slot {slot + 1}</label>
          <select id="quick-slot-{slot}" value={slotApp(quick, slot)} onchange={(e) => setSlot(slot, e.currentTarget.value)}>
            <option value="">{quick.Suggest ? "Suggested for you" : "Empty"}</option>
            {#each appChoices as app (app.ID)}
              <option value={app.ID}>{app.Name}</option>
            {/each}
          </select>
        </div>
      {/each}
      <label class="item">
        <span class="text">
          <span class="name">Suggest apps from how I use Omni</span>
          <span class="sub">
            Fills the open slots with the apps you usually open at this time of day. It is worked out on this {isMac ? "Mac" : "PC"} from
            when you opened apps through Omni, and nothing is sent anywhere.
          </span>
        </span>
        <input type="checkbox" class="switch" checked={quick.Suggest} onchange={(e) => saveQuick({ Suggest: e.currentTarget.checked })} />
      </label>
    {/if}
    <div class="item">
      <span class="text">
        <span class="name">Forget my usage history</span>
        <span class="sub">Clears the launch times Omni keeps for suggestions and ranking. Pins and tags stay.</span>
      </span>
      <button type="button" class="btn quiet" onclick={forgetUsage}>{forgotten ? "Forgotten" : "Forget"}</button>
    </div>
  </div>
{/if}

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
  .themes .custom {
    position: relative;
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
  .themes .custom:hover {
    background: var(--lift);
  }
  .themes .custom.on {
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
    color: var(--text);
  }
  .themes .custom:focus-within {
    outline: 1px solid var(--chrome);
    outline-offset: 2px;
  }
  /* The browser's own colour picker, laid invisibly over the chip. */
  .custom input {
    position: absolute;
    inset: 0;
    opacity: 0;
    cursor: pointer;
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
