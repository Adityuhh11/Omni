<script lang="ts">
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { SearchView, View } from "../../../bindings/omni/internal/settingsui";
  import { duration } from "../format";
  import Icon from "../Icon.svelte";
  import { keycaps } from "../keycaps";
  import { combo, isMac } from "../platform";
  import IndexWindow from "../settings/IndexWindow.svelte";
  import ProgressBar from "./ProgressBar.svelte";

  interface Props {
    view: View;
    search: SearchView;
    /** Whether the person set up file search in this guide. */
    files: boolean;
    refresh: () => Promise<void>;
    /** Goes back to the folders screen. */
    onfolders: () => void;
  }
  let { view, search, files, refresh, onfolders }: Props = $props();

  let error = $state("");

  const ready = $derived(search.Installed && search.Enabled && search.EngineState === "ready" && !search.NeedsUpdate);
  const percent = $derived(search.Total > 0 ? Math.min(100, Math.round((search.Done / search.Total) * 100)) : 0);
  const megabytes = (bytes: number) => (bytes / (1024 * 1024)).toFixed(0);
  const hasFolders = $derived((view.Roots ?? []).length > 0);
  // A first index of hours is worth offering the night for.
  const LONG = 2 * 3600;

  const KEYS: [string, string][] = [
    [combo("Tab"), "switches between everything, apps, files and the web"],
    [combo("Ctrl+K"), "shows what can be done with the selected row"],
    [combo("Ctrl+,"), "opens Settings"],
    [combo("Esc"), "closes the bar"],
  ];

  async function start() {
    error = await Settings.StartIndexing();
    await refresh();
  }
</script>

<div class="hero">
  <span class="seal"><Icon name="check" size={28} /></span>
  <h2>You are all set</h2>
  <p class="lede">
    Press {#each keycaps(view.Hotkey) as part (part)}<kbd>{part}</kbd> {/each} in any program to open the bar. Omni lives in the {isMac ? "menu bar" : "notification area, by the clock"};
    closing this window does not stop anything.
  </p>
</div>

<ul class="keys">
  {#each KEYS as [key, does] (key)}
    <li><kbd>{key}</kbd><span>{does}</span></li>
  {/each}
</ul>
<p class="help again">Open this guide again from the Omni icon, or in Settings, About.</p>

{#if !files}
  <p class="help">
    File search is off, so only apps, the web, the clipboard and snippets are searched. Turn it on any time in Settings > File search; it is
    the same download.
  </p>
{:else if search.Installing}
  <div class="card">
    <p class="label">Downloading file search</p>
    <ProgressBar percent={search.Total > 0 ? percent : null} label={search.StepName} />
    <p class="help">
      Step {search.Step} of {search.Steps}: {search.StepName}{search.Total > 0 ? ` · ${megabytes(search.Done)} of ${megabytes(search.Total)} MB` : ""}.
      Nothing is indexed until you start it, so you can use Omni meanwhile.
    </p>
  </div>
{:else if search.Error}
  <p class="error">The download failed: {search.Error}</p>
  <div><button type="button" class="btn primary" onclick={() => Settings.TurnOn().then(refresh)}>Try again</button></div>
{:else if search.EngineState === "failed"}
  <p class="error">The search engine could not start: {search.EngineMessage}</p>
{:else if !ready}
  <div class="card">
    <p class="label">Starting the search engine</p>
    <p class="help">A few seconds.</p>
  </div>
{:else if search.HoldIndexing}
  <div class="card">
    <p class="label">Ready to index</p>
    {#if hasFolders}
      <p class="help">
        {(view.Roots ?? []).length} folder{(view.Roots ?? []).length === 1 ? "" : "s"} will be read, newest files first, and you can search while it
        works. {search.Device === "cpu" ? "On the processor it takes a while; Gentle keeps the computer free." : ""}
      </p>
      {#if search.Estimate.Complete}
        <p class="help">
          Indexing everything takes {duration(search.Estimate.Seconds)} at this speed ({duration(search.Estimate.ByPace?.fast ?? 0)} at Fast,
          {duration(search.Estimate.ByPace?.gentle ?? 0)} at Gentle). The time left shows once it is running.
        </p>
      {:else}
        <p class="help">Counting what is in your folders to say how long it will take...</p>
      {/if}
      {#if search.Estimate.Complete && search.Estimate.Seconds > LONG}
        <p class="help">That is a long time. Omni can work only at night and leave the computer alone by day.</p>
        <IndexWindow value={search.IndexBetween} onsave={async (between) => { error = await Settings.SetIndexBetween(between); await refresh(); }} />
      {/if}
      <div><button type="button" class="btn primary" onclick={start}>Start indexing</button></div>
    {:else}
      <p class="help">No folder is chosen yet.</p>
      <div><button type="button" class="btn" onclick={onfolders}>Choose folders</button></div>
    {/if}
  </div>
{:else}
  <div class="card">
    <p class="label">{search.IndexState === "uptodate" ? "Everything is indexed" : "Reading your files"}</p>
    <p class="help">{search.Paused ? "Indexing is paused." : search.IndexText}</p>
    {#if search.IndexState === "indexing" && search.IndexTotal > 0}
      <ProgressBar percent={Math.round((search.IndexDone / search.IndexTotal) * 100)} label="Files indexed" />
    {/if}
    <p class="help">You can search right away: results fill in as Omni reads your files, newest first. The notification-area icon shows how it is going.</p>
  </div>
{/if}
{#if error}<p class="error">{error}</p>{/if}

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin: 4px 0 0;
    padding: 16px;
    border: 1px solid var(--line);
    border-radius: 12px;
    background: var(--tile);
  }
  .card .label {
    margin: 0;
    font: 600 13.5px/1.2 var(--sans);
  }
  .hero {
    display: flex;
    flex-direction: column;
    align-items: center;
    margin: 6px 0 22px;
    text-align: center;
  }
  /* The accent, lit, with a small spring as it lands. */
  .seal {
    display: grid;
    place-items: center;
    width: 60px;
    height: 60px;
    margin-bottom: 14px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--chrome) 16%, transparent);
    color: var(--chrome);
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, var(--chrome) 50%, transparent),
      0 0 34px -6px var(--chrome);
    animation: land 640ms var(--spring) both;
  }
  @keyframes land {
    from {
      opacity: 0;
      transform: scale(0.6);
    }
  }
  .hero h2 {
    font-size: 24px;
  }
  .hero .lede {
    max-width: 520px;
    margin-bottom: 0;
  }
  .keys {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    margin: 0 0 8px;
    padding: 0;
    list-style: none;
  }
  .keys li {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 11px 14px;
    border-radius: 14px;
    background: var(--tile);
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.35;
  }
  .keys kbd {
    flex: none;
    min-width: 46px;
  }
  .again {
    margin-bottom: 22px;
    text-align: center;
  }
  @media (prefers-reduced-motion: reduce) {
    .seal {
      animation: none;
    }
  }
</style>
