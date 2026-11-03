<script lang="ts">
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
    });
    if (error === "") {
      saved = true;
      clearTimeout(savedTimer);
      savedTimer = setTimeout(() => (saved = false), 1800);
    }
  }
</script>

<div class="head-row">
  <h2>Timer</h2>
  <p class="ok" aria-live="polite">{saved ? "Saved" : ""}</p>
</div>

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
</style>
