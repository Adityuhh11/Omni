<script lang="ts">
  import { onMount } from "svelte";
  import { cubicOut } from "svelte/easing";
  import { fly } from "svelte/transition";
  import { Window } from "@wailsio/runtime";
  import { OmniAI, Service as Settings } from "../bindings/omni/internal/settingsui";
  import type { OmniAIView, SearchView, View } from "../bindings/omni/internal/settingsui";
  import Card from "./lib/Card.svelte";
  import RoundButton from "./lib/RoundButton.svelte";
  import "./lib/settings/forms.css";
  import { after, before, stepsFor, type Step } from "./lib/setup/flow";
  import Done from "./lib/setup/Done.svelte";
  import Features from "./lib/setup/Features.svelte";
  import FilesStep from "./lib/setup/FilesStep.svelte";
  import Hotkey from "./lib/setup/Hotkey.svelte";
  import Look from "./lib/setup/Look.svelte";
  import Welcome from "./lib/setup/Welcome.svelte";

  let step = $state<Step>("welcome");
  // Which way the last move went, so a screen slides in from the side the person is heading to.
  let heading = $state(1);
  let view = $state<View | null>(null);
  let search = $state<SearchView | null>(null);
  let ai = $state<OmniAIView | null>(null);
  let fileError = $state("");
  let content = $state<HTMLElement>();

  // Whether file search is on or on its way: it adds the folders screen.
  const files = $derived(!!search && (search.Installing || (search.Installed && search.Enabled)));
  const steps = $derived(stepsFor(files));
  const at = $derived(Math.max(0, steps.indexOf(step)));
  const reduced = typeof matchMedia !== "undefined" && matchMedia("(prefers-reduced-motion: reduce)").matches;

  const refreshSearch = async () => {
    search = await Settings.Search();
  };
  const refreshAI = async () => {
    ai = await OmniAI.View();
  };
  const refresh = async () => {
    [view, search, ai] = await Promise.all([Settings.View(), Settings.Search(), OmniAI.View()]);
  };

  // While a download runs the guide asks every second how it is going; the footer shows it on every screen.
  onMount(() => {
    document.documentElement.dataset.surface = "card";
    refresh();
    const timer = setInterval(() => Promise.all([refreshSearch(), refreshAI()]), 1000);
    return () => clearInterval(timer);
  });

  // Once, as long as the person has not changed any choice: a computer with no graphics card gets the cheap ones (pictures
  // take seconds each there, sound 4 times real time) and one with a card also gets its sound searched (about a minute
  // per hour of audio). Every choice is one click away.
  let preset = false;
  $effect(() => {
    if (!search || preset || !search.Device || !search.HoldIndexing) return;
    if (search.PDFPages !== "auto" || search.Photos !== "content" || search.Videos !== "name" || search.Audio !== "name") return;
    preset = true;
    const choices = search.Device === "cpu" ? ["text", "name", "name", "name"] : ["auto", "content", "name", "content"];
    Settings.SetIndexing(choices[0], choices[1], choices[2], choices[3]).then(refreshSearch);
  });

  async function enable() {
    fileError = await Settings.TurnOn();
    await refreshSearch();
  }

  // Choosing where the download goes can be refused (a folder that cannot be written); the reason shows where the button is.
  async function chooseFolder(change: () => Promise<string>) {
    fileError = await change();
    await refreshSearch();
  }

  function go(next: Step) {
    heading = steps.indexOf(next) >= at ? 1 : -1;
    step = next;
    content?.scrollTo(0, 0);
  }
  const forward = () => (step === "done" ? Window.Close() : go(after(step, files)));

  // Enter moves on, unless it was meant for something that has its own use for it.
  function onkeydown(event: KeyboardEvent) {
    if (event.key !== "Enter" || event.defaultPrevented) return;
    if ((event.target as HTMLElement).closest("button, input, select, textarea, a, summary, [role='radio']")) return;
    forward();
  }

  const mb = (bytes: number) => (bytes / (1024 * 1024)).toFixed(0);
  const notes = $derived.by((): { text: string; problem: boolean }[] => {
    const out: { text: string; problem: boolean }[] = [];
    if (search && files) {
      if (search.Installing) {
        const part = search.Total > 0 ? ` · ${mb(search.Done)} of ${mb(search.Total)} MB` : "";
        out.push({ text: `Downloading file search · step ${search.Step} of ${search.Steps}${part}`, problem: false });
      } else if (search.Error) {
        out.push({ text: "The download failed: " + search.Error, problem: true });
      } else if (search.Installed && search.Enabled) {
        out.push({ text: search.EngineState === "ready" ? "File search is ready" : "Starting the search engine...", problem: false });
      }
    }
    if (ai?.Installing) {
      out.push({ text: `Downloading Omni AI${ai.Total > 0 ? ` · ${mb(ai.Done)} of ${mb(ai.Total)} MB` : ""}`, problem: false });
    }
    return out;
  });

  const LABELS: Partial<Record<Step, string>> = { welcome: "Get started", done: "Done" };
</script>

<svelte:window {onkeydown} />

<Card>
  <main class="setup">
    <header>
      <div class="drag" aria-hidden="true"></div>
      <ol class="pips" aria-label="Progress">
        {#each steps as s, i (s)}
          <li class:done={i < at} class:now={i === at} aria-current={i === at ? "step" : undefined}></li>
        {/each}
      </ol>
      <div class="close"><RoundButton icon="close" label="Close" onclick={() => Window.Close()} /></div>
    </header>

    <div class="content" bind:this={content}>
      {#if view && search}
        {#key step}
          <div class="screen" in:fly={{ x: reduced ? 0 : heading * 24, duration: reduced ? 0 : 300, easing: cubicOut }}>
            {#if step === "welcome"}
              <Welcome />
            {:else if step === "hotkey"}
              <Hotkey {view} />
            {:else if step === "look"}
              <Look {view} />
            {:else if step === "features"}
              <Features {search} {ai} {fileError} {refreshAI} onenable={enable} onchoose={() => chooseFolder(Settings.ChooseDataFolder)} onreset={() => chooseFolder(Settings.UseDefaultDataFolder)} />
            {:else if step === "files"}
              <FilesStep {search} {refresh} {refreshSearch} />
            {:else}
              <Done {view} {search} {files} {refresh} onfolders={() => go("files")} />
            {/if}
          </div>
        {/key}
      {/if}
    </div>

    <footer>
      <div class="status" aria-live="polite">
        {#each notes as note (note.text)}<span class:problem={note.problem}>{note.text}</span>{/each}
      </div>
      <div class="nav">
        {#if step !== "welcome"}
          <button type="button" class="btn" onclick={() => go(before(step, files))}>Back</button>
        {/if}
        <button type="button" class="btn primary" onclick={forward}>{LABELS[step] ?? "Continue"}</button>
      </div>
    </footer>
  </main>
</Card>

<style>
  .setup {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }
  header {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    flex: none;
    height: 54px;
  }
  /* The top of the window drags it; the close button sits above that strip. */
  .drag {
    position: absolute;
    inset: 0;
    --wails-draggable: drag;
  }
  .close {
    position: absolute;
    top: 10px;
    right: 14px;
    z-index: 2;
  }
  /* One pip a screen: those walked are lit, the one you are on is long. */
  .pips {
    position: relative;
    display: flex;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .pips li {
    width: 6px;
    height: 6px;
    border-radius: 3px;
    background: var(--lift-keyed);
    transition:
      width 360ms var(--ease),
      background 360ms var(--ease);
  }
  .pips li.done {
    background: color-mix(in srgb, var(--chrome) 50%, transparent);
  }
  .pips li.now {
    width: 20px;
    background: var(--chrome);
  }
  .content {
    flex: 1;
    min-height: 0;
    padding: 6px 32px 20px;
    overflow-y: auto;
  }
  /* Every screen is one column, so what is on it lines up from screen to screen. */
  .screen {
    max-width: 520px;
    margin: 0 auto;
  }
  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 60px;
    padding: 0 22px 0 32px;
    border-top: 1px solid var(--line);
  }
  .status {
    display: flex;
    flex-direction: column;
    gap: 2px;
    color: var(--text-3);
    font-size: 12px;
  }
  .status .problem {
    color: var(--danger);
  }
  .nav {
    display: flex;
    flex: none;
    gap: 8px;
  }
  @media (prefers-reduced-motion: reduce) {
    .pips li {
      transition: none;
    }
  }
</style>
