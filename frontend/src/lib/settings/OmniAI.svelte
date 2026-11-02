<script lang="ts">
  import { onMount } from "svelte";
  import { Browser } from "@wailsio/runtime";
  import { OmniAI } from "../../../bindings/omni/internal/settingsui";
  import type { OmniAIView } from "../../../bindings/omni/internal/settingsui";
  import { addressHost, toolHint, unloadsIn } from "../chat/chat";

  let page = $state<OmniAIView | null>(null);
  let error = $state("");
  let confirming = $state<"" | "remove" | "clear">("");
  let showCustom = $state(false);

  // What is typed in the section about another model; it starts from what is saved, and is not overwritten by the page's refreshes.
  let draft = $state({ url: "", model: "", key: "", loaded: false });
  let test = $state<{ busy: boolean; error: string; models: string[] } | null>(null);

  const refresh = async () => {
    page = await OmniAI.View();
    if (!draft.loaded) {
      draft.url = page.Custom.URL;
      draft.model = page.Custom.Model;
      draft.loaded = true;
    }
  };

  // While a download runs the page asks every second for its progress; otherwise now and then, for the model's state.
  onMount(() => {
    refresh();
    const timer = setInterval(refresh, 1000);
    return () => clearInterval(timer);
  });

  /** Runs a change, shows why it was refused, and shows the page as it is now. */
  async function run(action: () => Promise<string | void>) {
    error = (await action()) ?? "";
    await refresh();
  }

  const megabytes = (bytes: number) => (bytes / (1024 * 1024)).toFixed(bytes < 10 * 1024 * 1024 ? 1 : 0);
  const percent = $derived(page && page.Total > 0 ? Math.min(100, Math.round((page.Done / page.Total) * 100)) : 0);
  const where = $derived(addressHost(draft.url));
  const headline = $derived.by(() => {
    if (!page) return "";
    if (page.Custom.On) return page.Enabled ? "On" : "Off";
    if (page.Installing) return "Downloading";
    if (!page.Installed) return "Not installed";
    if (page.NeedsUpdate) return "Needs an update";
    return page.Enabled ? "On" : "Off";
  });
  const modelText = $derived.by(() => {
    if (!page) return "";
    if (page.Custom.On) return `Asking ${page.Custom.Model} at ${page.Custom.Host}.`;
    switch (page.ModelState) {
      case "ready":
        return `In memory, on ${page.Device === "cpu" ? "the processor" : "the graphics card"}${page.UnloadsAt ? `; ${unloadsIn(page.UnloadsAt)}` : ""}.`;
      case "loading":
        return "Starting the model…";
      case "failed":
        return page.ModelMessage || "The model did not start.";
      default:
        return page.ModelMessage || "Not in memory. It starts when you ask.";
    }
  });
  const loadLabels: Record<string, string> = { when_asked: "When I ask something (the model loads the first time)", at_start: "When Omni starts (answers right away, uses memory all day)" };
  const minutesLabel = (m: number) => (m === 0 ? "Never" : m === 1 ? "1 minute" : m < 60 ? `${m} minutes` : "1 hour");
  const daysLabel = (d: number) => (d === 0 ? "For good" : d === 365 ? "A year" : `${d} days`);
  const contextLabel = (n: number) => `${n.toLocaleString("en-US")} tokens${n === 8192 ? " (usual)" : ""}`;

  async function behaviour(change: { load?: string; minutes?: number; context?: number }) {
    if (!page) return;
    await run(() => OmniAI.SetBehaviour(change.load ?? page!.Load, change.minutes ?? page!.UnloadAfterMinutes, change.context ?? page!.ContextTokens));
  }

  async function history(change: { keep?: boolean; days?: number }) {
    if (!page) return;
    await run(() => OmniAI.SetHistory(change.keep ?? page!.KeepHistory, change.days ?? page!.HistoryDays));
  }

  async function testConnection() {
    test = { busy: true, error: "", models: [] };
    const answer = await OmniAI.TestCustom(draft.url, draft.key);
    test = { busy: false, error: answer.Error, models: answer.Models ?? [] };
    if (!answer.Error && !draft.model && answer.Models?.length === 1) draft.model = answer.Models[0];
  }

  async function useCustom() {
    if (draft.key.trim()) {
      error = await OmniAI.SetCustomKey(draft.key);
      if (error) return;
      draft.key = "";
    }
    await run(() => OmniAI.UseCustom(draft.url, draft.model));
  }

  async function remove() {
    confirming = "";
    await run(() => OmniAI.Remove());
  }

  async function clear() {
    confirming = "";
    await run(() => OmniAI.ClearHistory());
  }
</script>

<h2>Omni AI</h2>
<p class="lede">
  Ask questions in a small chat window.
  {#if page?.Custom.On && !page.Custom.Local}
    The model you chose answers, at {page.Custom.Host}: what you write goes there.
  {:else if page?.Custom.On}
    The model you chose answers, on this computer: nothing you write leaves it.
  {:else}
    A language model answers on this computer: nothing you write leaves it.
  {/if}
  Open it by typing <span class="mono">ask</span> in the bar.
</p>

{#if page}
  <section class="block">
    <header class="top">
      <span class="dot" class:on={page.Enabled && page.ModelState === "ready"} class:bad={page.ModelState === "failed" || !!page.Error} aria-hidden="true"></span>
      <div class="who">
        <h3>{headline}</h3>
        {#if page.Custom.On}<p class="help">{page.Enabled ? modelText : "Off."}</p>
        {:else if page.Installed && !page.Installing && !page.NeedsUpdate}<p class="help">{page.Enabled ? modelText : "Installed and off."}</p>{/if}
      </div>
      {#if page.Custom.On}
        <button type="button" class="btn" class:primary={!page.Enabled} onclick={() => run(() => (page!.Enabled ? OmniAI.TurnOff() : OmniAI.TurnOn()))}>{page.Enabled ? "Turn off" : "Turn on"}</button>
      {:else if page.Installing}
        <button type="button" class="btn" onclick={() => run(() => OmniAI.CancelInstall())}>Cancel</button>
      {:else if !page.Installed}
        <button type="button" class="btn primary" onclick={() => run(() => OmniAI.TurnOn())}>Download and turn on</button>
      {:else if page.NeedsUpdate}
        <button type="button" class="btn primary" onclick={() => run(() => OmniAI.Update())}>Update</button>
      {:else if page.Enabled}
        <button type="button" class="btn" onclick={() => run(() => OmniAI.TurnOff())}>Turn off</button>
      {:else}
        <button type="button" class="btn primary" onclick={() => run(() => OmniAI.TurnOn())}>Turn on</button>
      {/if}
    </header>

    {#if page.Custom.On}
      <!-- nothing to download or load: that server is not Omni's -->
    {:else if page.Installing}
      <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={percent} aria-label={page.StepName}>
        <div class="fill" style:transform="scaleX({(page.Total > 0 ? percent : 4) / 100})"></div>
      </div>
      <p class="help">
        Step {page.Step} of {page.Steps}: {page.StepName}{#if page.Total > 0}, {megabytes(page.Done)} of {megabytes(page.Total)} MB{/if}. You can close this
        window; the download goes on.
      </p>
    {:else if !page.Installed}
      <p class="help">
        A one-time download of about {(page.DownloadMB / 1024).toFixed(1)} GB from {(page.Hosts ?? []).join(" and ")}: the model (Liquid AI's LFM3-1.7B) and the
        program that runs it. Nothing is downloaded until you press the button.
        <button type="button" class="link" onclick={() => Browser.OpenURL(page!.LicenseURL)}>Read the model's licence</button>
      </p>
    {:else if page.NeedsUpdate}
      <p class="help">This version of Omni comes with a newer model server. Updating keeps your chats.</p>
    {:else if page.Enabled && page.ModelState === "ready"}
      <div><button type="button" class="btn" onclick={() => run(() => OmniAI.Unload())}>Unload now</button></div>
    {/if}
    {#if page.Error}<p class="error">The last attempt failed: {page.Error}</p>{/if}
    {#if error}<p class="error">{error}</p>{/if}
  </section>

  <section class="block">
    <h3>Which model</h3>
    <div class="field">
      <label class="check"><input type="radio" name="ai-source" checked={!page.Custom.On && !showCustom} onchange={() => { showCustom = false; run(() => OmniAI.UseBundled()); }} /> The one Omni downloads: private, it runs on this computer</label>
      <label class="check"><input type="radio" name="ai-source" checked={page.Custom.On || showCustom} onchange={() => (showCustom = true)} /> Another model: LM Studio or Ollama on this computer, or a service with an API key</label>
    </div>
    {#if page.Custom.On || showCustom}
      <div class="field">
        <label class="label" for="ai-url">Address</label>
        <input id="ai-url" type="text" class="mono" placeholder="http://localhost:1234/v1" bind:value={draft.url} spellcheck="false" />
        <p class="help">LM Studio: <span class="mono">http://localhost:1234/v1</span>. Ollama: <span class="mono">http://localhost:11434/v1</span>. OpenAI: <span class="mono">https://api.openai.com/v1</span>.</p>
      </div>
      <div class="field">
        <label class="label" for="ai-key">API key</label>
        <input id="ai-key" type="password" class="mono" placeholder={page.Custom.HasKey ? "Saved; type a new one to replace it" : "Only if the service asks for one"} bind:value={draft.key} autocomplete="off" spellcheck="false" />
        <p class="help">
          Kept in the system credential store, never in a file.
          {#if page.Custom.HasKey}<button type="button" class="link" onclick={() => run(() => OmniAI.SetCustomKey(""))}>Forget the saved key</button>{/if}
        </p>
      </div>
      <div class="field">
        <label class="label" for="ai-model">Model</label>
        <input id="ai-model" type="text" class="mono" list="ai-models" placeholder="the model's name, as the server calls it" bind:value={draft.model} spellcheck="false" />
        <datalist id="ai-models">{#each test?.models ?? [] as name}<option value={name}></option>{/each}</datalist>
      </div>
      {#if draft.url.trim() && !where.local}
        <p class="warn">
          {where.host ? `Everything you ask, and anything the tools give the model (your files, notes, clipboard, browser history, if you switched those on), is sent to ${where.host}.` : "That is not an address."}
          Its terms and its charges apply, not Omni's.
        </p>
      {:else if where.local}
        <p class="help">This address is on this computer: nothing you ask leaves it.</p>
      {/if}
      <div class="row">
        <span class="help">
          {#if test?.busy}Asking the server…{:else if test?.error}{test.error}{:else if test}Connected. {test.models.length} model{test.models.length === 1 ? "" : "s"} offered.{/if}
        </span>
        <button type="button" class="btn" disabled={!draft.url.trim() || test?.busy} onclick={testConnection}>Test the connection</button>
        <button type="button" class="btn primary" disabled={!draft.url.trim() || !draft.model.trim()} onclick={useCustom}>{page.Custom.On ? "Save" : "Use this model"}</button>
      </div>
    {/if}
  </section>

  {#if !page.Custom.On}
  <section class="block">
    <h3>When the model is in memory</h3>
    <div class="field">
      <label class="label" for="ai-load">Load the model</label>
      <select id="ai-load" value={page.Load} onchange={(e) => behaviour({ load: e.currentTarget.value })}>
        {#each page.Loads ?? [] as choice}<option value={choice}>{loadLabels[choice] ?? choice}</option>{/each}
      </select>
    </div>
    <div class="field">
      <label class="label" for="ai-unload">Unload it after</label>
      <select id="ai-unload" value={String(page.UnloadAfterMinutes)} onchange={(e) => behaviour({ minutes: Number(e.currentTarget.value) })}>
        {#each page.UnloadChoices ?? [] as minutes}<option value={String(minutes)}>{minutesLabel(minutes)}</option>{/each}
      </select>
      <p class="help">Counted from the end of the last answer. Unloading frees the graphics memory (about 1.2 GB); the next question starts the model again, which takes a few seconds.</p>
    </div>
    <div class="field">
      <label class="label" for="ai-context">How much it can read at once</label>
      <select id="ai-context" value={String(page.ContextTokens)} onchange={(e) => behaviour({ context: Number(e.currentTarget.value) })}>
        {#each page.ContextChoices ?? [] as tokens}<option value={String(tokens)}>{contextLabel(tokens)}</option>{/each}
      </select>
      <p class="help">A conversation longer than this forgets its oldest messages (the chat shows where). Bigger costs little memory, but a small model reads long text less well.</p>
    </div>
    <div class="field">
      <label class="check"><input type="checkbox" checked={page.Thinking} onchange={(e) => run(() => OmniAI.SetThinking(e.currentTarget.checked))} /> Let it think before it answers</label>
      <p class="help">Slower, and sometimes better for puzzles and code. The chat window has a Think button for the same switch.</p>
    </div>
    <div class="field">
      <label class="check"><input type="checkbox" checked={page.OnTop} onchange={(e) => run(() => OmniAI.SetOnTop(e.currentTarget.checked))} /> Keep the chat above other windows</label>
      <p class="help">Off, other windows can cover it and it comes back when you open it again.</p>
    </div>
  </section>
  {/if}

  <section class="block">
    <h3>Tools</h3>
    <p class="help">
      What Omni AI may use on this computer while it answers. Each is off until you switch it on, here or from the + in the chat.
      {#if page.Custom.On && !page.Custom.Local}
        What a tool finds is given to the model, so it goes to {page.Custom.Host} with your question.
      {:else}
        Nothing leaves this computer for them; with web search also on, the model could put what a tool found into the words it searches for.
      {/if}
    </p>
    {#each page.Tools ?? [] as tool (tool.group)}
      <div class="field">
        <div class="line">
          <label class="check"><input type="checkbox" checked={tool.on} onchange={(e) => run(() => OmniAI.SetTool(tool.group, e.currentTarget.checked))} /> {tool.title}</label>
          {#if tool.writes}
            <select
              class="ask"
              aria-label="When {tool.title} changes something"
              value={tool.always ? "run" : "ask"}
              disabled={!tool.on}
              onchange={(e) => run(() => OmniAI.SetAlwaysRun(tool.group, e.currentTarget.value === "run"))}
            >
              <option value="ask">Ask me first</option>
              <option value="run">Run without asking</option>
            </select>
          {/if}
        </div>
        <p class="help">{toolHint(tool)}</p>
      </div>
    {/each}
    <div class="row">
      <span class="help">
        A script in <span class="mono">{page.ScriptsFolder}</span> with an <span class="mono">@omni.tool</span> line becomes a tool here. See SCRIPTS.md in Omni's
        documentation.
      </span>
      <button type="button" class="btn quiet" onclick={() => run(() => OmniAI.OpenScriptsFolder())}>Show the folder</button>
    </div>
  </section>

  <section class="block">
    <h3>Past chats</h3>
    <div class="field">
      <label class="check"><input type="checkbox" checked={page.KeepHistory} onchange={(e) => history({ keep: e.currentTarget.checked })} /> Keep my chats</label>
      <p class="help">Saved as plain files in <span class="mono">{page.HistoryFolder}</span>, not encrypted. Saving sends nothing anywhere. A single chat can be marked "Not saved" in its window.</p>
    </div>
    <div class="field">
      <label class="label" for="ai-days">Delete chats I have not opened for</label>
      <select id="ai-days" value={String(page.HistoryDays)} disabled={!page.KeepHistory} onchange={(e) => history({ days: Number(e.currentTarget.value) })}>
        {#each page.HistoryChoices ?? [] as days}<option value={String(days)}>{daysLabel(days)}</option>{/each}
      </select>
    </div>
    <div class="row">
      <span class="help">{page.Conversations === 1 ? "1 chat" : `${page.Conversations} chats`} saved.</span>
      {#if confirming === "clear"}
        <button type="button" class="btn danger" onclick={clear}>Delete them all</button>
        <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
      {:else}
        <button type="button" class="btn" disabled={page.Conversations === 0} onclick={() => (confirming = "clear")}>Clear history</button>
        <button type="button" class="btn quiet" onclick={() => run(() => OmniAI.OpenHistoryFolder())}>Show the folder</button>
      {/if}
    </div>
  </section>

  {#if page.Installed && !page.Installing}
    <section class="block remove">
      <div class="row">
        {#if confirming === "remove"}
          <button type="button" class="btn danger" onclick={remove}>Delete the model (about 1 GB)</button>
          <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
        {:else}
          <button type="button" class="btn quiet" onclick={() => (confirming = "remove")}>Remove the model…</button>
        {/if}
      </div>
      <p class="help">Turns Omni AI off and deletes the model and its program. Your chats stay.</p>
    </section>
  {/if}
{/if}

<style>
  .block {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 22px 0;
    border-top: 1px solid var(--line);
  }
  .block:first-of-type {
    padding-top: 0;
    border-top: 0;
  }
  .block h3 {
    margin: 0;
    font: 600 13.5px/1.3 var(--sans);
  }
  .block .field {
    margin-bottom: 8px;
  }
  .top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .who {
    flex: 1;
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
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .row .help {
    margin-right: auto;
  }
  .remove {
    padding-bottom: 0;
  }
  .remove .row {
    margin-left: -14px; /* the quiet buttons' own padding, so their words line up with the page */
  }
  .bar {
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
  .link {
    padding: 0;
    border: 0;
    background: none;
    color: var(--text-2);
    font: inherit;
    text-decoration: underline;
    text-decoration-color: var(--text-3);
    text-underline-offset: 2px;
    cursor: pointer;
  }
  .link:hover {
    color: var(--text);
  }
  .line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .ask {
    width: auto;
    height: 30px;
  }
  .warn {
    margin: 0;
    padding: 10px 12px;
    border: 1px solid color-mix(in srgb, var(--danger) 40%, transparent);
    border-radius: 10px;
    color: var(--text);
    font: 12.5px/1.45 var(--sans);
  }
</style>
