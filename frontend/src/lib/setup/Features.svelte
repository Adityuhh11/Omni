<script lang="ts">
  import { onMount, type ComponentProps } from "svelte";
  import { OmniAI, Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { ClipboardView, OmniAIView, SearchView } from "../../../bindings/omni/internal/settingsui";
  import Icon from "../Icon.svelte";
  import FileDetails from "./FileDetails.svelte";
  import Offer from "./Offer.svelte";
  import ProgressBar from "./ProgressBar.svelte";

  interface Props {
    search: SearchView;
    ai: OmniAIView | null;
    /** Why the last attempt to start the file-search download was refused, or "". */
    fileError: string;
    /** Asks the guide to read Omni AI's state again after a change. */
    refreshAI: () => Promise<void>;
    onenable: () => void;
    onchoose: () => void;
    onreset: () => void;
  }
  let { search, ai, fileError, refreshAI, onenable, onchoose, onreset }: Props = $props();

  // What works the moment Omni runs: named so nobody has to find it later.
  const READY: { icon: ComponentProps<typeof Icon>["name"]; name: string }[] = [
    { icon: "app", name: "Apps" },
    { icon: "web", name: "Web" },
    { icon: "note", name: "Notes" },
    { icon: "timer", name: "Timer" },
    { icon: "calc", name: "Calculator" },
    { icon: "snippet", name: "Snippets" },
  ];

  // Clipboard history: a switch that saves as it is flipped, like everywhere in Settings.
  let clip = $state<ClipboardView | null>(null);
  let clipOn = $state(false);
  let clipError = $state("");
  onMount(async () => {
    clip = await Settings.Clipboard();
    clipOn = clip.Enabled;
  });
  async function saveClip() {
    if (!clip) return;
    clipError = await Settings.SaveClipboard(clipOn, clip.Ignore ?? [], clip.Hotkey);
    if (clipError !== "") clipOn = !clipOn;
  }

  // Omni AI: a download the person starts, then a switch.
  let aiError = $state("");
  const needsDownload = $derived(!!ai && !ai.Custom.On && (!ai.Installed || ai.NeedsUpdate));
  async function runAI(action: () => Promise<string | void>) {
    aiError = (await action()) ?? "";
    await refreshAI();
  }

  // File search: the details and the cost come first, and nothing starts until the button in them is pressed.
  let details = $state(false);
  const filesStarted = $derived(search.Installing || (search.Installed && search.Enabled));
  const percent = (done: number, total: number) => (total > 0 ? Math.min(100, Math.round((done / total) * 100)) : null);
  const gb = (mb: number) => (mb / 1024).toFixed(1);
</script>

<h2>What should Omni do for you?</h2>
<p class="lede">Most of it works already. Switch on the extras you want; Settings changes any of it later.</p>

<h3 class="group-title">Ready to use</h3>
<ul class="ready">
  {#each READY as r (r.name)}
    <li><Icon name={r.icon} size={15} />{r.name}</li>
  {/each}
</ul>

<h3 class="group-title">Extras</h3>
<div class="group">
  <Offer icon="clip" title="Clipboard history" blurb="Search what you copied (clip:). Kept on this computer." on={clipOn} showDetail={clipError !== ""}>
    {#snippet control()}
      <input type="checkbox" class="switch" aria-label="Clipboard history" bind:checked={clipOn} onchange={saveClip} disabled={!clip} />
    {/snippet}
    {#snippet detail()}
      {#if clipError}<p class="error">{clipError}</p>{/if}
    {/snippet}
  </Offer>

  {#if ai}
    <Offer icon="chat" title="Omni AI" blurb="Chat with a model that runs on this computer (ask:)." on={ai.Enabled || ai.Installing} showDetail={ai.Installing || !!ai.Error || aiError !== ""}>
      {#snippet control()}
        {#if ai.Installing}
          <button type="button" class="btn quiet" onclick={() => runAI(() => OmniAI.CancelInstall())}>Cancel</button>
        {:else if needsDownload}
          <button type="button" class="btn" onclick={() => runAI(() => OmniAI.TurnOn())}>Download {gb(ai.DownloadMB)} GB</button>
        {:else}
          <input
            type="checkbox"
            class="switch"
            aria-label="Omni AI"
            checked={ai.Enabled}
            onchange={(e) => runAI(e.currentTarget.checked ? () => OmniAI.TurnOn() : () => OmniAI.TurnOff())}
          />
        {/if}
      {/snippet}
      {#snippet detail()}
        {#if ai.Installing}
          <ProgressBar percent={percent(ai.Done, ai.Total)} label="Downloading Omni AI" />
          <p class="help">Step {ai.Step} of {ai.Steps}. It switches on by itself when done; carry on meanwhile.</p>
        {/if}
        {#if ai.Error}<p class="error">The download failed: {ai.Error}</p>{/if}
        {#if aiError}<p class="error">{aiError}</p>{/if}
      {/snippet}
    </Offer>
  {/if}

  <Offer icon="search" title="File search" blurb="Find files by what is in them. About {gb(search.DownloadMB)} GB, once." on={filesStarted} showDetail={filesStarted || details}>
    {#snippet control()}
      {#if search.Installing}
        <span class="state">Downloading</span>
      {:else if filesStarted}
        <span class="state on"><Icon name="check" size={15} />On</span>
      {:else}
        <button type="button" class="btn" aria-expanded={details} onclick={() => (details = !details)}>{details ? "Hide" : "Set up"}</button>
      {/if}
    {/snippet}
    {#snippet detail()}
      {#if search.Installing}
        <ProgressBar percent={percent(search.Done, search.Total)} label={search.StepName} />
        <p class="help">Step {search.Step} of {search.Steps}. Carry on; the next screen picks the folders.</p>
      {:else if filesStarted}
        <p class="help">Installed. The next screen picks the folders; nothing is read until you start it.</p>
      {:else if details}
        <FileDetails {search} error={fileError} {onenable} onskip={() => (details = false)} {onchoose} {onreset} />
      {/if}
    {/snippet}
  </Offer>
</div>

<p class="help more">More in Settings (<kbd>Ctrl+,</kbd>): calendars, quicklinks, snippets, a hotkey for any app.</p>

<style>
  /* What works already, as small chips: it is a list to glance at, not to read. */
  .ready {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .ready li {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 11px 6px 9px;
    border-radius: 999px;
    background: var(--tile);
    color: var(--text-2);
    font: 560 12px/1 var(--sans);
  }
  .ready li :global(svg) {
    color: var(--chrome);
  }
  .group :global(.offer + .offer) {
    border-top: 1px solid var(--line);
  }
  .state {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--text-3);
    font: 560 12.5px/1 var(--sans);
  }
  .state.on {
    color: var(--chrome);
  }
  .more {
    margin-top: 16px;
  }
</style>
