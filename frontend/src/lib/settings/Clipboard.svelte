<script lang="ts">
  import { isMac } from "../platform";
  import { onMount } from "svelte";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { ClipboardView } from "../../../bindings/omni/internal/settingsui";

  let page = $state<ClipboardView | null>(null);
  let enabled = $state(false);
  let ignore = $state("");
  let error = $state("");
  let saved = $state(false);
  let clearing = $state(false);
  let savedTimer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    page = await Settings.Clipboard();
    enabled = page.Enabled;
    ignore = (page.Ignore ?? []).join("\n");
  }
  onMount(load);

  function flashSaved() {
    saved = true;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => (saved = false), 1800);
  }

  async function save() {
    error = await Settings.SaveClipboard(enabled, ignore.split("\n"), page?.Hotkey ?? "");
    if (error !== "") {
      enabled = page?.Enabled ?? false;
      return;
    }
    await load();
    flashSaved();
  }

  async function clear() {
    clearing = false;
    error = await Settings.ClearClipboard();
    await load();
  }
</script>

<div class="head-row">
  <h2>Clipboard</h2>
  <p class="ok" aria-live="polite">{saved ? "Saved" : ""}</p>
</div>

{#if page}
  <h3 class="group-title">History</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Remember what I copy</span>
        <span class="sub">Text only: the newest 500 copies from 30 days, kept on this computer. Copies marked private are never kept.</span>
      </span>
      <input type="checkbox" class="switch" bind:checked={enabled} onchange={save} />
    </label>
    <div class="item">
      <span class="text">
        <span class="name">{page.Count === 0 ? "Nothing kept" : page.Count === 1 ? "1 copy kept" : page.Count + " copies kept"}</span>
        <span class="sub mono">{page.File}</span>
      </span>
      <span class="end">
        {#if clearing}
          <button type="button" class="btn danger" onclick={clear}>Clear history</button>
          <button type="button" class="btn quiet" onclick={() => (clearing = false)}>Keep</button>
        {:else}
          <button type="button" class="btn" disabled={page.Count === 0} onclick={() => (clearing = true)}>Clear…</button>
        {/if}
      </span>
    </div>
  </div>
  <p class="group-note">In the bar, type <span class="mono">clip:</span> and a few words; Enter pastes it where you were.</p>

  <h3 class="group-title">Programs to leave out</h3>
  <div class="group">
    <div class="item stack">
      <textarea id="ignore" aria-label="Programs to leave out" rows="4" bind:value={ignore} onchange={save} spellcheck="false" placeholder={isMac ? "One app per line, for example MyVault" : "One program per line, for example MyVault.exe"}></textarea>
      <span class="sub">Always left out: {(page.Defaults ?? []).join(", ")}</span>
    </div>
  </div>

  {#if isMac}
    <p class="group-note">
      Pasting into the app you were in needs the Accessibility permission (System Settings, Privacy &amp; Security, Accessibility). Without it Enter copies, and you press <kbd>⌘V</kbd>.
    </p>
  {/if}

  {#if error}<p class="error">{error}</p>{/if}
{/if}

<style>
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
  textarea::placeholder {
    color: var(--text-3);
  }
  .sub.mono {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11.5px;
  }
  label.item {
    cursor: pointer;
  }
</style>
