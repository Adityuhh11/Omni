<script lang="ts">
  import { isMac } from "../platform";
  import { onDestroy, onMount } from "svelte";
  import { Clipboard } from "@wailsio/runtime";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { UpdateProgress, View } from "../../../bindings/omni/internal/settingsui";

  let { view }: { view: View } = $props();

  let copied = $state(false);
  let error = $state("");
  // svelte-ignore state_referenced_locally
  let found = $state(view.UpdateFound);
  // svelte-ignore state_referenced_locally
  let installable = $state(view.UpdateInstallable);
  // svelte-ignore state_referenced_locally
  let daily = $state(view.CheckUpdates);
  let checking = $state(false);
  let outcome = $state("");
  let progress = $state<UpdateProgress | null>(null);
  let pollTimer: ReturnType<typeof setTimeout> | undefined;
  const busy = $derived(progress?.Stage === "downloading" || progress?.Stage === "installing" || progress?.Stage === "done");

  // While an update installs the page asks how it is going; it stops asking when it is done or has failed.
  async function poll() {
    clearTimeout(pollTimer);
    progress = await Settings.UpdateProgress();
    if (progress.Stage === "downloading" || progress.Stage === "installing") pollTimer = setTimeout(poll, 400);
  }
  onMount(poll);
  onDestroy(() => clearTimeout(pollTimer));

  async function startUpdate() {
    error = await Settings.StartUpdate();
    if (error === "") poll();
  }

  const megabytes = (bytes: number) => (bytes / 1048576).toFixed(1);
  const stageText = $derived.by(() => {
    if (!progress) return "";
    if (progress.Stage === "downloading") {
      return progress.Total > 0 ? `Downloading ${megabytes(progress.Done)} of ${megabytes(progress.Total)} MB` : "Downloading";
    }
    if (progress.Stage === "installing") return "Checking and installing";
    return progress.Error === "" ? "Installed. The program restarts now." : "";
  });

  async function check() {
    checking = true;
    outcome = "";
    const r = await Settings.CheckUpdate();
    checking = false;
    if (r.Error) {
      outcome = "Could not check: " + r.Error;
      return;
    }
    found = r.Available ? r.Latest : "";
    installable = r.Installable;
    outcome = r.Available ? "" : "No release is newer than " + view.Version + ".";
  }

  async function saveDaily() {
    error = await Settings.SetCheckUpdates(daily);
    if (error !== "") daily = !daily;
    else if (daily) check();
  }

  async function copyDiagnostics() {
    await Clipboard.SetText(await Settings.Diagnostics());
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }
</script>

<h2>About</h2>

<h3 class="group-title">Omni {view.Version}</h3>
<div class="group">
  <div class="item">
    <span class="text">
      <span class="name">{#if found}Version {found} is available{:else}Updates{/if}</span>
      {#if busy}
        <span class="sub" aria-live="polite">{stageText}</span>
      {:else if found && !installable}
        <span class="sub">
          {#if isMac}Run the install line from the release page in Terminal; it keeps your settings and index.{:else}This release has no download that can be checked.{/if}
        </span>
      {:else if outcome}
        <span class="sub">{outcome}</span>
      {/if}
    </span>
    <span class="end">
      {#if found && busy}
        {#if progress?.Stage === "downloading"}<button type="button" class="btn quiet" onclick={() => Settings.CancelUpdate()}>Cancel</button>{/if}
      {:else if found}
        <button type="button" class="btn" class:quiet={installable} onclick={async () => (error = await Settings.OpenUpdate())}>Release page</button>
        {#if installable}<button type="button" class="btn primary" onclick={startUpdate}>Update</button>{/if}
      {:else}
        <button type="button" class="btn" disabled={checking} onclick={check}>{checking ? "Checking…" : "Check now"}</button>
      {/if}
    </span>
  </div>
  {#if found && progress?.Stage === "downloading" && progress.Total > 0}
    <div class="item"><progress max={progress.Total} value={progress.Done}></progress></div>
  {/if}
  {#if found && progress?.Stage === "done" && progress.Error}<p class="item error">{progress.Error}</p>{/if}
  {#if found && !busy && progress?.Stage === "failed"}<p class="item error">{progress.Error} Try again, or download it from the release page.</p>{/if}
  <label class="item">
    <span class="text">
      <span class="name">Check once a day</span>
      <span class="sub">Asks api.github.com for the latest version number, nothing else</span>
    </span>
    <input type="checkbox" class="switch" bind:checked={daily} onchange={saveDaily} />
  </label>
</div>

<h3 class="group-title">Help</h3>
<div class="group">
  <div class="item">
    <span class="name">Setup guide</span>
    <button type="button" class="btn" onclick={() => Settings.OpenSetup()}>Open</button>
  </div>
  <div class="item">
    <span class="text">
      <span class="name">Diagnostics</span>
      <span class="sub">Version, {isMac ? "macOS" : "Windows"}, file search state. No file or folder names.</span>
    </span>
    <button type="button" class="btn" onclick={copyDiagnostics}>{copied ? "Copied" : "Copy"}</button>
  </div>
</div>

<h3 class="group-title">Files</h3>
<div class="group">
  <div class="item">
    <span class="text">
      <span class="name">Settings</span>
      <span class="sub mono">{view.ConfigPath}</span>
    </span>
    <button type="button" class="btn" onclick={async () => (error = await Settings.OpenFolder("config"))}>Show</button>
  </div>
  <div class="item">
    <span class="text">
      <span class="name">Log</span>
      <span class="sub mono">{view.LogDir}</span>
    </span>
    <button type="button" class="btn" onclick={async () => (error = await Settings.OpenFolder("logs"))}>Show</button>
  </div>
</div>
{#if error}<p class="error">{error}</p>{/if}

<style>
  progress {
    width: 100%;
    height: 6px;
    accent-color: var(--chrome);
  }
  .item.error {
    min-height: 0;
    font-size: 12px;
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
