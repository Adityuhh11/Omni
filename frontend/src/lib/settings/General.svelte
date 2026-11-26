<script lang="ts">
  import { isMac, words } from "../platform";
  import { onMount } from "svelte";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { AppChoice, View } from "../../../bindings/omni/internal/settingsui";
  import type { QuickOpen } from "../../../bindings/omni/internal/config";
  import { applyLook } from "../theme";
  import LookPicker from "./LookPicker.svelte";

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

  async function saveRing() {
    applyLook({ Ring: ring });
    error = await Settings.SetRing(ring);
    if (error !== "") applyLook({ Ring: (ring = !ring) });
    else flashSaved();
  }

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
  <LookPicker bind:theme bind:accent onsaved={flashSaved} />
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
  label.item {
    cursor: pointer;
  }
</style>
