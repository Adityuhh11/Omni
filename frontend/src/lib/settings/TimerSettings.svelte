<script lang="ts">
  import { isMac } from "../platform";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { View } from "../../../bindings/omni/internal/settingsui";

  let { view }: { view: View } = $props();

  // The form starts from the settings as they were when the page opened and then owns its own values.
  // svelte-ignore state_referenced_locally
  let focus = $state(view.Timer.FocusMinutes);
  // svelte-ignore state_referenced_locally
  let short = $state(view.Timer.ShortBreakMinutes);
  // svelte-ignore state_referenced_locally
  let long = $state(view.Timer.LongBreakMinutes);
  // svelte-ignore state_referenced_locally
  let rounds = $state(view.Timer.Rounds);
  // svelte-ignore state_referenced_locally
  let auto = $state(view.Timer.AutoNext);
  // svelte-ignore state_referenced_locally
  let presets = $state((view.Timer.PresetMinutes ?? []).join(", "));
  // svelte-ignore state_referenced_locally
  let block = $state(view.Timer.Block);
  // svelte-ignore state_referenced_locally
  let sites = $state((view.Timer.BlockSites ?? []).join("\n"));
  // svelte-ignore state_referenced_locally
  let apps = $state((view.Timer.BlockApps ?? []).join("\n"));
  const lines = (text: string) => text.split("\n").map((l) => l.trim()).filter(Boolean);
  let error = $state("");
  let saved = $state(false);
  let savedTimer: ReturnType<typeof setTimeout> | undefined;

  async function save() {
    const parts = presets.split(/[,\s]+/).filter(Boolean);
    const minutes = parts.map(Number);
    if (minutes.some((m) => !Number.isInteger(m))) {
      error = `“${parts.find((p) => !Number.isInteger(Number(p)))}” is not a whole number of minutes.`;
      return;
    }
    error = await Settings.SaveTimer({
      FocusMinutes: focus,
      ShortBreakMinutes: short,
      LongBreakMinutes: long,
      Rounds: rounds,
      AutoNext: auto,
      PresetMinutes: minutes,
      Block: block,
      BlockSites: lines(sites),
      BlockApps: lines(apps),
    });
    if (error === "") {
      saved = true;
      clearTimeout(savedTimer);
      savedTimer = setTimeout(() => (saved = false), 1800);
    }
  }
</script>

<div class="head-row">
  <h2>Focus &amp; timer</h2>
  <p class="ok" aria-live="polite">{saved ? "Saved" : ""}</p>
</div>

<h3 class="group-title">Focus</h3>
<div class="group">
  <label class="item">
    <span class="text">
      <span class="name">Block distractions while I focus</span>
      <span class="sub">During a focus period, a listed site or app is {isMac ? "hidden" : "minimized"} and the timer shows</span>
    </span>
    <input type="checkbox" class="switch" bind:checked={block} onchange={save} />
  </label>
  <div class="item stack">
    <label class="name" for="block-sites">Sites</label>
    <textarea id="block-sites" rows="4" bind:value={sites} onchange={save} disabled={!block} spellcheck="false" placeholder={"youtube.com\nreddit.com\nx.com"}></textarea>
  </div>
  <div class="item stack">
    <label class="name" for="block-apps">Apps</label>
    <textarea id="block-apps" rows="3" bind:value={apps} onchange={save} disabled={!block} spellcheck="false" placeholder={isMac ? "Discord\nSteam" : "discord.exe\nsteam.exe"}></textarea>
  </div>
</div>
<p class="group-note">
  Type <span class="mono">focus</span> or <span class="mono">focus 50</span> in the bar; a pomodoro's focus periods block too.
  {#if isMac}The first time, macOS asks whether Omni may read each browser's address. Firefox and Zen do not say which page they show, so for them only the apps list applies.{/if}
</p>

<h3 class="group-title">Pomodoro</h3>
<div class="group">
  <div class="item">
    <label class="name" for="focus">Focus</label>
    <span class="unit"><input id="focus" type="number" min="1" max="180" bind:value={focus} onchange={save} /> min</span>
  </div>
  <div class="item">
    <label class="name" for="short">Short break</label>
    <span class="unit"><input id="short" type="number" min="1" max="60" bind:value={short} onchange={save} /> min</span>
  </div>
  <div class="item">
    <label class="name" for="long">Long break</label>
    <span class="unit"><input id="long" type="number" min="1" max="120" bind:value={long} onchange={save} /> min</span>
  </div>
  <div class="item">
    <label class="name" for="rounds">Focus periods before the long break</label>
    <span class="unit"><input id="rounds" type="number" min="1" max="12" bind:value={rounds} onchange={save} /><span class="pad" aria-hidden="true">min</span></span>
  </div>
  <label class="item">
    <span class="text">
      <span class="name">Start the next period by itself</span>
      <span class="sub">A sound plays when a period ends either way</span>
    </span>
    <input type="checkbox" class="switch" bind:checked={auto} onchange={save} />
  </label>
</div>
<p class="group-note">Type <span class="mono">pomodoro</span> in the bar.</p>

<h3 class="group-title">Countdown</h3>
<div class="group">
  <div class="item">
    <span class="text">
      <label class="name" for="presets">Quick timers</label>
      <span class="sub">Up to 8, separated by commas</span>
    </span>
    <span class="unit"><input id="presets" type="text" class="presets" bind:value={presets} onchange={save} spellcheck="false" autocomplete="off" /> min</span>
  </div>
</div>
<p class="group-note">Any other length: <span class="mono">timer 17</span>, <span class="mono">timer 1h30</span>, <span class="mono">timer 90s tea</span>.</p>

{#if error}<p class="error">{error}</p>{/if}

<style>
  .unit {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-3);
    font-size: 12.5px;
  }
  input[type="number"] {
    width: 72px;
  }
  .pad {
    visibility: hidden; /* keeps the field in line with the ones that have a unit */
  }
  .item input.presets {
    width: 160px;
  }
  label.item {
    cursor: pointer;
  }
  textarea {
    box-sizing: border-box;
    width: 100%;
    padding: 9px 12px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--tile);
    color: var(--text);
    font: 450 13px/1.5 var(--mono);
    resize: vertical;
  }
  textarea:focus {
    outline: none;
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
  }
  textarea:disabled {
    color: var(--text-3);
  }
  textarea::placeholder {
    color: var(--text-3);
  }
</style>
