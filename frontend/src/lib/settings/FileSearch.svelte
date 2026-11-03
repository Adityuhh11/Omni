<script lang="ts">
  import { onMount } from "svelte";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { SearchView, View } from "../../../bindings/omni/internal/settingsui";
  import { changes, effects, type Choices, type Kind } from "../indexingPlan";
  import { CHOICES, SPEEDS } from "./choices";
  import { choiceHints, paceHints, surveyLine } from "./hints";
  import IndexWindow from "./IndexWindow.svelte";
  import ChoiceGroup from "./ChoiceGroup.svelte";
  import { draft } from "./draft";

  let { view, reload }: { view: View; reload: () => Promise<void> } = $props();

  let search = $state<SearchView | null>(null);
  let error = $state("");
  let confirming = $state("");
  let removingFiles = $state(false);
  let removingAll = $state(false);
  let optionsOpen = $state(false);
  const roots = $derived(view.Roots ?? []);

  const folderText = $derived(Object.fromEntries((search?.Folders ?? []).map((f) => [f.Root, f.Text])));
  const refused = $derived(new Set((search?.Folders ?? []).filter((f) => f.Blocked).map((f) => f.Root)));
  const holds = $derived(Object.fromEntries((search?.Surveys ?? []).map((f) => [f.Root, surveyLine(f)])));

  // On an index that exists a click only changes the draft; "Save and re-index" is what starts the work. Before anything
  // is indexed (right after the download, or with no folders yet) there is nothing to protect, so a click applies at once.
  const staged = $derived(!!search && search.Indexed > 0 && !search.HoldIndexing);
  const saved = $derived<Choices | null>(search ? { PDFPages: search.PDFPages, Photos: search.Photos, Videos: search.Videos, Audio: search.Audio } : null);
  const shown = $derived($draft ?? saved);
  const plan = $derived(saved && $draft ? changes(saved, $draft) : []);

  async function choose(kind: Kind, id: string) {
    if (!saved) return;
    const next = { ...(shown ?? saved), [kind]: id };
    if (!staged) {
      await run(() => Settings.SetIndexing(next.PDFPages, next.Photos, next.Videos, next.Audio));
      return;
    }
    $draft = changes(saved, next).length > 0 ? next : null;
  }

  async function saveDraft() {
    if (!$draft) return;
    const next = $draft;
    $draft = null;
    await run(() => Settings.SetIndexing(next.PDFPages, next.Photos, next.Videos, next.Audio));
  }

  const refresh = async () => {
    search = await Settings.Search();
  };

  // While an install runs the page asks every second for its progress; otherwise now and then, for the engine's state.
  onMount(() => {
    refresh();
    const timer = setInterval(refresh, 1000);
    return () => clearInterval(timer);
  });

  const megabytes = (bytes: number) => (bytes / (1024 * 1024)).toFixed(bytes < 10 * 1024 * 1024 ? 1 : 0);
  const percent = $derived(search && search.Total > 0 ? Math.min(100, Math.round((search.Done / search.Total) * 100)) : 0);
  const engineText = $derived.by(() => {
    if (!search) return "";
    switch (search.EngineState) {
      case "ready":
        return "Running. Search works while the index builds.";
      case "starting":
      case "warming":
        return "Starting…";
      case "restarting":
        return `Restarting: ${search.EngineMessage}`;
      case "failed":
        return `It could not start: ${search.EngineMessage}`;
      default:
        return "Off";
    }
  });

  async function run(action: () => Promise<string>) {
    error = await action();
    await refresh();
  }

  async function add() {
    error = await Settings.AddFolder();
    await reload();
  }

  async function remove(folder: string) {
    confirming = "";
    error = await Settings.RemoveFolder(folder);
    await reload();
  }

  async function removeEverything() {
    removingAll = false;
    $draft = null;
    await run(() => Settings.RemoveEverything());
    await reload();
  }

  // The options are folded away until they matter: before the first indexing, or with a change waiting to be saved.
  $effect(() => {
    if (search?.HoldIndexing || plan.length > 0) optionsOpen = true;
  });

  const running = $derived(!!search && search.Enabled && search.Installed && !search.NeedsUpdate && search.EngineState === "ready");
  const failed = $derived(!!search && (search.EngineState === "failed" || !!search.Error));
  const headline = $derived.by(() => {
    if (!search) return "";
    if (search.Installing) return "Downloading";
    if (!search.Installed) return "Not installed";
    if (search.NeedsUpdate) return "Needs an update";
    return search.Enabled ? "On" : "Off";
  });

  async function removeFiles() {
    removingFiles = false;
    await run(() => Settings.RemoveFiles());
  }
</script>

<h2>File search</h2>

{#if search}
  <h3 class="group-title">Finds files by what they contain. It runs on this computer; nothing leaves it</h3>
  <div class="group">
    <div class="item">
      <span class="who">
        <span class="dot" class:on={running} class:bad={failed} aria-hidden="true"></span>
        <span class="text">
          <span class="name">{headline}</span>
          {#if search.Installed && !search.Installing && !search.NeedsUpdate}<span class="sub">{search.Enabled ? engineText : "Installed and off"}</span>{/if}
        </span>
      </span>
      {#if search.Installing}
        <button type="button" class="btn" onclick={() => Settings.CancelInstall().then(refresh)}>Cancel</button>
      {:else if !search.Installed}
        <button type="button" class="btn primary" onclick={() => run(() => Settings.TurnOn())}>Download and turn on</button>
      {:else if search.NeedsUpdate}
        <button type="button" class="btn primary" onclick={() => run(() => Settings.Update())}>Update</button>
      {:else if search.Enabled}
        <button type="button" class="btn" onclick={() => run(() => Settings.TurnOff())}>Turn off</button>
      {:else}
        <button type="button" class="btn primary" onclick={() => run(() => Settings.TurnOn())}>Turn on</button>
      {/if}
    </div>

    {#if search.Installing}
      <div class="item stack">
        <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={percent} aria-label={search.StepName}>
          <div class="fill" style:transform="scaleX({(search.Total > 0 ? percent : 4) / 100})"></div>
        </div>
        <span class="sub">
          Step {search.Step} of {search.Steps}: {search.StepName}{#if search.Total > 0}, {megabytes(search.Done)} of {megabytes(search.Total)} MB{/if}. It goes on if you close this window.
        </span>
      </div>
    {:else if !search.Installed}
      <p class="item sub">About {(search.DownloadMB / 1024).toFixed(1)} GB, once, from {(search.Hosts ?? []).join(", ")}. Nothing is indexed until you say so.</p>
      <div class="item">
        <span class="text">
          <span class="name">Saved in</span>
          <span class="sub mono path">{search.Data.Engine}</span>
        </span>
        {#if search.Data.CanChange}
          <span class="end">
            {#if search.Data.Chosen}<button type="button" class="btn quiet" onclick={() => run(() => Settings.UseDefaultDataFolder())}>Use the usual place</button>{/if}
            <button type="button" class="btn" onclick={() => run(() => Settings.ChooseDataFolder())}>Change…</button>
          </span>
        {/if}
      </div>
    {:else if search.NeedsUpdate}
      <p class="item sub">This version of Omni comes with a newer search engine. Updating keeps your index.</p>
    {:else if search.Enabled && search.EngineState === "ready" && !search.HoldIndexing}
      <div class="item">
        <span class="sub">{search.Paused ? "Indexing is paused" : search.IndexText}</span>
        {#if search.IndexState === "indexing"}
          <button type="button" class="btn" onclick={() => (search?.Paused ? Settings.ResumeIndexing() : Settings.PauseIndexing()).then(refresh)}>
            {search.Paused ? "Resume" : "Pause"}
          </button>
        {/if}
      </div>
      {#if search.IndexState === "indexing" && search.IndexTotal > 0}
        <div class="item thin">
          <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax={search.IndexTotal} aria-valuenow={search.IndexDone} aria-label="Files indexed in this folder">
            <div class="fill" style:transform="scaleX({search.IndexDone / search.IndexTotal})"></div>
          </div>
        </div>
      {/if}
    {/if}
    {#if search.Error}<p class="item error">The last attempt failed: {search.Error}</p>{/if}
  </div>

  {#if search.Enabled && search.EngineState === "ready" && search.HoldIndexing && !search.NeedsUpdate}
    <div class="group callout" role="status">
      <div class="item">
        <span class="text">
          <span class="name">Ready. Nothing is indexed yet</span>
          <span class="sub">
            {search.Device === "cpu" ? "On the processor here, so pictures are slow." : "On the graphics card."} Add folders, check the options, then start.
          </span>
        </span>
        <button type="button" class="btn primary" disabled={roots.length === 0} onclick={() => run(() => Settings.StartIndexing())}>Start indexing</button>
      </div>
    </div>
  {/if}
{/if}

<h3 class="group-title">Folders</h3>
<div class="group">
  {#each roots as root (root)}
    <div class="item">
      <span class="text">
        <span class="path mono" title={root}>{root}</span>
        {#if folderText[root]}<span class="sub">{folderText[root]}</span>{/if}
        {#if holds[root]}<span class="sub">{holds[root]}</span>{/if}
      </span>
      <span class="end">
        {#if refused.has(root)}
          <button type="button" class="btn quiet" onclick={() => Settings.OpenFileAccessSettings()}>Open System Settings</button>
        {/if}
        {#if confirming === root}
          <button type="button" class="btn danger" onclick={() => remove(root)}>Remove</button>
          <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
        {:else}
          <button type="button" class="btn quiet" onclick={() => (confirming = root)}>Remove</button>
        {/if}
      </span>
    </div>
  {:else}
    <p class="item sub">No folders yet. Nothing is searched until you add one.</p>
  {/each}
  <div class="item">
    <button type="button" class="btn" onclick={add}>Add folder…</button>
  </div>
</div>
{#if error}<p class="error">{error}</p>{/if}

{#if search?.Installed}
  <h3 class="group-title">In the bar</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Include files in the default search</span>
        <span class="sub">Off, Tab switches to Files</span>
      </span>
      <input type="checkbox" class="switch" checked={search.InLauncher} onchange={(e) => run(() => Settings.SetInLauncher(e.currentTarget.checked))} />
    </label>
    <label class="item">
      <span class="text">
        <span class="name">Group files by type</span>
        <span class="sub">Best match first, then Code, Documents, PDFs, Images, Videos, Audio</span>
      </span>
      <input type="checkbox" class="switch" checked={search.GroupResults} onchange={(e) => run(() => Settings.SetGroupResults(e.currentTarget.checked))} />
    </label>
  </div>

  <details bind:open={optionsOpen}>
    <summary class="group-title">Indexing options</summary>
    <div class="group">
      {#if search.Device === "cpu"}
        <p class="item warn">No usable graphics card: each picture takes about 5 seconds. Words only and names are the quick choices.</p>
      {/if}
      {#each CHOICES as choice (choice.kind)}
        <div class="item stack">
          <span class="name" id="{choice.kind}-label">{choice.label}</span>
          <ChoiceGroup labelId="{choice.kind}-label" options={choice.options} value={shown?.[choice.kind]} onpick={(id) => choose(choice.kind, id)} hints={choiceHints(choice.kind, choice.options.map((o) => o.id), search.Estimate)} />
        </div>
      {/each}
      {#if plan.length > 0}
        <div class="item stack draft" role="status">
          <span class="name">Not saved yet</span>
          <ul>
            {#each effects(plan, search.Device) as line (line)}<li>{line}</li>{/each}
          </ul>
          <span class="end">
            <button type="button" class="btn primary" onclick={saveDraft}>Save and re-index</button>
            <button type="button" class="btn quiet" onclick={() => ($draft = null)}>Discard</button>
          </span>
        </div>
      {/if}
      <div class="item stack">
        <span class="name" id="speed-label">How hard it works</span>
        <ChoiceGroup labelId="speed-label" options={SPEEDS} value={search.Pace} onpick={(id) => run(() => Settings.SetPace(id))} hints={paceHints(search.Estimate)} />
      </div>
      <label class="item">
        <span class="text">
          <span class="name">Wait for power</span>
          <span class="sub">Indexing pauses on battery. Searching still works.</span>
        </span>
        <input type="checkbox" class="switch" checked={search.PauseOnBattery} onchange={(e) => run(() => Settings.SetPauseOnBattery(e.currentTarget.checked))} />
      </label>
      <div class="item stack">
        <IndexWindow value={search.IndexBetween} onsave={(between) => run(() => Settings.SetIndexBetween(between))} />
      </div>
    </div>
  </details>
{/if}

{#if search && !search.Installing}
  <h3 class="group-title">Storage</h3>
  <div class="group">
    {#if search.Installed}
      <div class="item">
        <span class="text">
          <span class="name">Delete the downloaded files</span>
          <span class="sub">The engine and models. The index stays and works again after a new download.</span>
        </span>
        <span class="end">
          {#if removingFiles}
            <button type="button" class="btn danger" onclick={removeFiles}>Delete</button>
            <button type="button" class="btn quiet" onclick={() => (removingFiles = false)}>Keep</button>
          {:else}
            <button type="button" class="btn" onclick={() => (removingFiles = true)}>Delete…</button>
          {/if}
        </span>
      </div>
    {/if}
    <div class="item">
      <span class="text">
        <span class="name">Remove everything</span>
        <span class="sub">The engine, models and index; forgets the folders and options; turns file search off.</span>
      </span>
      <span class="end">
        {#if removingAll}
          <button type="button" class="btn danger" onclick={removeEverything}>Delete everything</button>
          <button type="button" class="btn quiet" onclick={() => (removingAll = false)}>Keep</button>
        {:else}
          <button type="button" class="btn danger" onclick={() => (removingAll = true)}>Remove…</button>
        {/if}
      </span>
    </div>
  </div>
{/if}

<style>
  .who {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }
  .dot {
    flex: none;
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: var(--text-3);
  }
  .dot.on {
    background: var(--chrome);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--chrome) 15%, transparent);
  }
  .dot.bad {
    background: var(--danger);
  }
  .callout {
    margin-top: 12px;
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
    background: var(--lift);
  }
  .bar {
    flex: 1;
    height: 6px;
    border-radius: 3px;
    background: var(--tile);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    border-radius: 3px;
    background: var(--chrome);
    transform-origin: left;
    transition: transform 0.4s var(--ease);
  }
  .item.thin {
    min-height: 0;
    padding-top: 0;
    border-top: 0;
  }
  .item.sub,
  .item.error {
    min-height: 0;
    font-size: 12px;
  }
  .warn {
    margin: 0;
    color: var(--text);
    font: 12.5px/1.45 var(--sans);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
  }
  details > summary {
    list-style: none;
    cursor: pointer;
  }
  details > summary::-webkit-details-marker {
    display: none;
  }
  details > summary::before {
    content: "";
    display: inline-block;
    width: 5px;
    height: 5px;
    margin: 0 8px 2px 1px;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: rotate(-45deg);
    transition: transform 0.12s;
  }
  details[open] > summary::before {
    transform: rotate(45deg);
  }
  details > summary:hover {
    color: var(--text-2);
  }
  .draft {
    background: var(--lift);
  }
  .draft ul {
    margin: 0;
    padding-left: 18px;
    color: var(--text-2);
    font-size: 12px;
    line-height: 1.5;
  }
  .draft .end {
    justify-content: flex-start;
  }
  .path {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl; /* a long path keeps its end, where the folder's own name is */
    text-align: left;
    color: var(--text-2);
  }
  label.item {
    cursor: pointer;
  }
  @media (prefers-reduced-motion: reduce) {
    .fill,
    details > summary::before {
      transition: none;
    }
  }
</style>
