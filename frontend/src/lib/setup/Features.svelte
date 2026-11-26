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

  // What works the moment Omni runs: shown so nobody has to find it later.
  const READY: { icon: ComponentProps<typeof Icon>["name"]; name: string; eg: string }[] = [
    { icon: "app", name: "Apps", eg: "vsc" },
    { icon: "web", name: "Web and quicklinks", eg: "gh rust" },
    { icon: "note", name: "Notes", eg: "note:" },
    { icon: "timer", name: "Timer and focus", eg: "timer 25" },
    { icon: "calc", name: "Calculator", eg: "12*7+3" },
    { icon: "snippet", name: "Snippets", eg: "snip:" },
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
<p class="lede">Most of it works already. Switch on the extras you want; every one can be changed later in Settings.</p>

<h3 class="group-title">Ready to use</h3>
<ul class="ready">
  {#each READY as r (r.name)}
    <li>
      <span class="tile"><Icon name={r.icon} size={18} /></span>
      <span class="what"><strong>{r.name}</strong><kbd>{r.eg}</kbd></span>
    </li>
  {/each}
</ul>

<h3 class="group-title">Switch on what you want</h3>
<div class="offers">
  <Offer icon="clip" title="Clipboard history" blurb="Find anything you copied in the last 30 days. Text only, kept on this computer; password managers are skipped." example="clip: invoice" on={clipOn} showDetail={clipError !== ""}>
    {#snippet control()}
      <input type="checkbox" class="switch" aria-label="Clipboard history" bind:checked={clipOn} onchange={saveClip} disabled={!clip} />
    {/snippet}
    {#snippet detail()}
      {#if clipError}<p class="error">{clipError}</p>{/if}
    {/snippet}
  </Offer>

  {#if ai}
    <Offer icon="chat" title="Omni AI" blurb="Chat with a small language model that runs on this computer. It can use your files, notes and calendar when you allow it." example="ask:" on={ai.Enabled || ai.Installing} showDetail={ai.Installing || !!ai.Error || aiError !== ""}>
      {#snippet control()}
        {#if ai.Installing}
          <button type="button" class="btn quiet" onclick={() => runAI(() => OmniAI.CancelInstall())}>Cancel</button>
        {:else if needsDownload}
          <button type="button" class="btn primary" onclick={() => runAI(() => OmniAI.TurnOn())}>Download · {gb(ai.DownloadMB)} GB</button>
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
          <p class="help">
            Downloading, once: step {ai.Step} of {ai.Steps}. Omni AI switches itself on when it is done; carry on meanwhile.
          </p>
        {/if}
        {#if ai.Error}<p class="error">The download failed: {ai.Error}</p>{/if}
        {#if aiError}<p class="error">{aiError}</p>{/if}
      {/snippet}
    </Offer>
  {/if}

  <Offer
    icon="search"
    title="File search"
    blurb="Find a file by what is in it, not by its name: a page of a PDF, a function, a photo, the moment in a video. About {gb(search.DownloadMB)} GB, once."
    example="the diagram with three boxes"
    on={filesStarted}
    showDetail={filesStarted || details}
  >
    {#snippet control()}
      {#if search.Installing}
        <span class="state">Downloading</span>
      {:else if filesStarted}
        <span class="state on"><Icon name="check" size={15} />On</span>
      {:else}
        <button type="button" class="btn" aria-expanded={details} onclick={() => (details = !details)}>{details ? "Hide" : "Set up..."}</button>
      {/if}
    {/snippet}
    {#snippet detail()}
      {#if search.Installing}
        <ProgressBar percent={percent(search.Done, search.Total)} label={search.StepName} />
        <p class="help">Downloading, once: step {search.Step} of {search.Steps}. Carry on; the next screen picks the folders.</p>
      {:else if filesStarted}
        <p class="help">Installed. The next screen picks the folders to read; nothing is read until you start it.</p>
      {:else if details}
        <FileDetails {search} error={fileError} {onenable} onskip={() => (details = false)} {onchoose} {onreset} />
      {/if}
    {/snippet}
  </Offer>
</div>

<p class="help more">More in Settings (<kbd>Ctrl+,</kbd>): calendars, quicklinks, snippets, a hotkey for any app, and the sites to keep out of the way while you focus.</p>

<style>
  .ready {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 0 0 22px;
    padding: 0;
    list-style: none;
  }
  .ready li {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    padding: 10px 12px;
    border-radius: 14px;
    background: var(--tile);
  }
  .tile {
    display: grid;
    place-items: center;
    flex: none;
    width: 34px;
    height: 34px;
    border-radius: 10px;
    background: var(--lift);
    color: var(--chrome);
  }
  .what {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    min-width: 0;
  }
  .what strong {
    font: 600 12.5px/1.2 var(--sans);
  }
  .what kbd {
    padding: 1px 6px;
    font-size: 11px;
    color: var(--text-2);
  }
  .offers {
    display: flex;
    flex-direction: column;
    gap: 10px;
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
    margin-top: 18px;
  }
</style>
