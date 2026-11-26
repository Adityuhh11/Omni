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
  const loadLabels: Record<string, string> = { when_asked: "When I ask", at_start: "When Omni starts" };
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

  // The Add script dialog: null while closed. The page's refreshes leave it alone.
  const OUTPUTS = [
    ["silent", "Do nothing with what it prints"],
    ["copy", "Copy what it prints"],
    ["paste", "Type what it prints into the program I was in"],
  ];
  let script = $state<{ name: string; ext: string; output: string; body: string; error: string } | null>(null);
  let addedScript = $state("");

  function startScript() {
    addedScript = "";
    script = { name: "", ext: page?.ScriptLanguages?.[0]?.Ext ?? "", output: "silent", body: "", error: "" };
  }

  /** Fills the code box from a file, and suggests its name and language when the dialog has none yet. */
  async function loadScript() {
    if (!script) return;
    const file = await OmniAI.LoadScript();
    if (file.Error) script.error = file.Error;
    else if (file.Text !== "") {
      script.body = file.Text;
      script.name ||= file.Name;
      if (file.Ext) script.ext = file.Ext;
      script.error = "";
    }
  }

  async function addScript() {
    if (!script) return;
    script.error = await OmniAI.AddScript(script.name, script.ext, script.output, script.body);
    if (script.error) return;
    addedScript = script.name.trim();
    script = null;
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

{#if page}
  <h3 class="group-title">
    {#if page.Custom.On && !page.Custom.Local}Answers come from {page.Custom.Host}: what you write goes there{:else}Answers on this computer: nothing you write leaves it{/if}
  </h3>
  <div class="group">
    <div class="item">
      <span class="who">
        <span class="dot" class:on={page.Enabled && page.ModelState === "ready"} class:bad={page.ModelState === "failed" || !!page.Error} aria-hidden="true"></span>
        <span class="text">
          <span class="name">{headline}</span>
          {#if page.Custom.On}<span class="sub">{page.Enabled ? modelText : "Off"}</span>
          {:else if page.Installed && !page.Installing && !page.NeedsUpdate}<span class="sub">{page.Enabled ? modelText : "Installed and off"}</span>{/if}
        </span>
      </span>
      <span class="end">
        {#if !page.Custom.On && page.Enabled && page.ModelState === "ready" && page.Installed && !page.Installing && !page.NeedsUpdate}
          <button type="button" class="btn quiet" onclick={() => run(() => OmniAI.Unload())}>Unload</button>
        {/if}
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
      </span>
    </div>
    {#if !page.Custom.On && page.Installing}
      <div class="item stack">
        <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={percent} aria-label={page.StepName}>
          <div class="fill" style:transform="scaleX({(page.Total > 0 ? percent : 4) / 100})"></div>
        </div>
        <span class="sub">
          Step {page.Step} of {page.Steps}: {page.StepName}{#if page.Total > 0}, {megabytes(page.Done)} of {megabytes(page.Total)} MB{/if}. It goes on if you close this window.
        </span>
      </div>
    {:else if !page.Custom.On && !page.Installed}
      <p class="item sub">
        <span>
          About {(page.DownloadMB / 1024).toFixed(1)} GB, once, from {(page.Hosts ?? []).join(" and ")}: Liquid AI's LFM3-1.7B and the program that runs it.
          <button type="button" class="link" onclick={() => Browser.OpenURL(page!.LicenseURL)}>Licence</button>
        </span>
      </p>
    {:else if !page.Custom.On && page.NeedsUpdate}
      <p class="item sub">This version of Omni comes with a newer model server. Updating keeps your chats.</p>
    {/if}
    {#if page.Error}<p class="item error">The last attempt failed: {page.Error}</p>{/if}
    {#if error}<p class="item error">{error}</p>{/if}
  </div>
  <p class="group-note">Type <span class="mono">ask</span> in the bar.</p>

  <h3 class="group-title">Model</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Omni's own</span>
        <span class="sub">Downloaded once, runs on this computer</span>
      </span>
      <input type="radio" name="ai-source" checked={!page.Custom.On && !showCustom} onchange={() => { showCustom = false; run(() => OmniAI.UseBundled()); }} />
    </label>
    <label class="item">
      <span class="text">
        <span class="name">Another model</span>
        <span class="sub">LM Studio or Ollama here, or a service with an API key</span>
      </span>
      <input type="radio" name="ai-source" checked={page.Custom.On || showCustom} onchange={() => (showCustom = true)} />
    </label>
    {#if page.Custom.On || showCustom}
      <div class="item">
        <span class="text">
          <label class="name" for="ai-url">Address</label>
          <span class="sub">LM Studio <span class="mono">:1234/v1</span>, Ollama <span class="mono">:11434/v1</span></span>
        </span>
        <input id="ai-url" type="text" class="mono" placeholder="http://localhost:1234/v1" bind:value={draft.url} spellcheck="false" />
      </div>
      <div class="item">
        <span class="text">
          <label class="name" for="ai-key">API key</label>
          <span class="sub">
            In the system credential store, never a file
            {#if page.Custom.HasKey}· <button type="button" class="link" onclick={() => run(() => OmniAI.SetCustomKey(""))}>Forget it</button>{/if}
          </span>
        </span>
        <input id="ai-key" type="password" class="mono" placeholder={page.Custom.HasKey ? "Saved; type to replace" : "Only if the service asks"} bind:value={draft.key} autocomplete="off" spellcheck="false" />
      </div>
      <div class="item">
        <label class="name" for="ai-model">Model</label>
        <input id="ai-model" type="text" class="mono" list="ai-models" placeholder="as the server names it" bind:value={draft.model} spellcheck="false" />
        <datalist id="ai-models">{#each test?.models ?? [] as name}<option value={name}></option>{/each}</datalist>
      </div>
      {#if draft.url.trim() && !where.local}
        <p class="item warn">
          {where.host ? `Everything you ask, and anything a tool you switched on finds (files, notes, clipboard, browser history), is sent to ${where.host}.` : "That is not an address."}
          Its terms and charges apply.
        </p>
      {/if}
      <div class="item">
        <span class="sub">
          {#if test?.busy}Asking the server…{:else if test?.error}{test.error}{:else if test}Connected, {test.models.length} model{test.models.length === 1 ? "" : "s"}{:else if where.local}On this computer: nothing leaves it{/if}
        </span>
        <span class="end">
          <button type="button" class="btn" disabled={!draft.url.trim() || test?.busy} onclick={testConnection}>Test</button>
          <button type="button" class="btn primary" disabled={!draft.url.trim() || !draft.model.trim()} onclick={useCustom}>{page.Custom.On ? "Save" : "Use this model"}</button>
        </span>
      </div>
    {/if}
  </div>

  <h3 class="group-title">Behaviour</h3>
  <div class="group">
    {#if !page.Custom.On}
      <div class="item">
        <label class="name" for="ai-load">Load the model</label>
        <select id="ai-load" value={page.Load} onchange={(e) => behaviour({ load: e.currentTarget.value })}>
          {#each page.Loads ?? [] as choice}<option value={choice}>{loadLabels[choice] ?? choice}</option>{/each}
        </select>
      </div>
      <div class="item">
        <span class="text">
          <label class="name" for="ai-unload">Unload it after</label>
          <span class="sub">Frees about 1.2 GB of graphics memory</span>
        </span>
        <select id="ai-unload" value={String(page.UnloadAfterMinutes)} onchange={(e) => behaviour({ minutes: Number(e.currentTarget.value) })}>
          {#each page.UnloadChoices ?? [] as minutes}<option value={String(minutes)}>{minutesLabel(minutes)}</option>{/each}
        </select>
      </div>
      <div class="item">
        <span class="text">
          <label class="name" for="ai-context">Context</label>
          <span class="sub">Longer chats forget their oldest messages</span>
        </span>
        <select id="ai-context" value={String(page.ContextTokens)} onchange={(e) => behaviour({ context: Number(e.currentTarget.value) })}>
          {#each page.ContextChoices ?? [] as tokens}<option value={String(tokens)}>{contextLabel(tokens)}</option>{/each}
        </select>
      </div>
      <label class="item">
        <span class="text">
          <span class="name">Think before answering</span>
          <span class="sub">Slower; sometimes better for puzzles and code</span>
        </span>
        <input type="checkbox" class="switch" checked={page.Thinking} onchange={(e) => run(() => OmniAI.SetThinking(e.currentTarget.checked))} />
      </label>
    {/if}
    <label class="item">
      <span class="name">Keep the chat above other windows</span>
      <input type="checkbox" class="switch" checked={page.OnTop} onchange={(e) => run(() => OmniAI.SetOnTop(e.currentTarget.checked))} />
    </label>
  </div>

  <h3 class="group-title">Tools</h3>
  <div class="group">
    {#each page.Tools ?? [] as tool (tool.group)}
      <div class="item">
        <span class="text">
          <span class="name">{tool.title}</span>
          <span class="sub">{toolHint(tool)}</span>
        </span>
        <span class="end">
          {#if tool.writes}
            <select
              class="ask"
              aria-label="When {tool.title} changes something"
              value={tool.always ? "run" : "ask"}
              disabled={!tool.on}
              onchange={(e) => run(() => OmniAI.SetAlwaysRun(tool.group, e.currentTarget.value === "run"))}
            >
              <option value="ask">Ask first</option>
              <option value="run">Don't ask</option>
            </select>
          {/if}
          <input type="checkbox" class="switch" aria-label={tool.title} checked={tool.on} onchange={(e) => run(() => OmniAI.SetTool(tool.group, e.currentTarget.checked))} />
        </span>
      </div>
    {/each}
    <div class="item">
      <span class="text">
        <span class="name">Your scripts</span>
        <span class="sub">A script with an <span class="mono">@omni.tool</span> line becomes a tool (SCRIPTS.md)</span>
      </span>
      <span class="end">
        <button type="button" class="btn" onclick={startScript}>Add script</button>
        <button type="button" class="btn quiet" onclick={() => run(() => OmniAI.OpenScriptsFolder())}>Show folder</button>
      </span>
    </div>
    {#if script}
      <div class="item stack" role="group" aria-label="Add a script">
        <div class="top">
          <input type="text" aria-label="Name" placeholder="Name, for example Kill port" bind:value={script.name} />
          <select class="ask" aria-label="Language" bind:value={script.ext}>
            {#each page.ScriptLanguages ?? [] as language (language.Ext)}<option value={language.Ext}>{language.Label}</option>{/each}
          </select>
        </div>
        <textarea aria-label="Code" rows="8" placeholder="The code it runs. Type it here, or load a file." bind:value={script.body} spellcheck="false"></textarea>
        <div class="top">
          <select class="ask" aria-label="What to do with its output" bind:value={script.output}>
            {#each OUTPUTS as [id, label] (id)}<option value={id}>{label}</option>{/each}
          </select>
          <button type="button" class="btn quiet" onclick={loadScript}>Load from file…</button>
        </div>
        {#if script.error}<p class="error" role="alert">{script.error}</p>{/if}
        <span class="end">
          <button type="button" class="btn primary" disabled={!script.name.trim() || !script.body.trim()} onclick={addScript}>Add</button>
          <button type="button" class="btn quiet" onclick={() => (script = null)}>Cancel</button>
        </span>
      </div>
    {:else if addedScript}
      <p class="item sub" role="status">Added “{addedScript}”. Type its name in the bar to run it.</p>
    {/if}
  </div>
  <p class="group-note">
    {#if page.Custom.On && !page.Custom.Local}
      What a tool finds goes to {page.Custom.Host} with your question.
    {:else}
      Tools send nothing anywhere; with web search on, the model could put what a tool found into its search words.
    {/if}
  </p>

  <h3 class="group-title">Past chats</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Keep my chats</span>
        <span class="sub">Plain, unencrypted files in <span class="mono">{page.HistoryFolder}</span></span>
      </span>
      <input type="checkbox" class="switch" checked={page.KeepHistory} onchange={(e) => history({ keep: e.currentTarget.checked })} />
    </label>
    <div class="item">
      <label class="name" for="ai-days">Delete chats not opened for</label>
      <select id="ai-days" value={String(page.HistoryDays)} disabled={!page.KeepHistory} onchange={(e) => history({ days: Number(e.currentTarget.value) })}>
        {#each page.HistoryChoices ?? [] as days}<option value={String(days)}>{daysLabel(days)}</option>{/each}
      </select>
    </div>
    <div class="item">
      <span class="name">{page.Conversations === 1 ? "1 chat" : `${page.Conversations} chats`} saved</span>
      <span class="end">
        {#if confirming === "clear"}
          <button type="button" class="btn danger" onclick={clear}>Delete them all</button>
          <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
        {:else}
          <button type="button" class="btn quiet" onclick={() => run(() => OmniAI.OpenHistoryFolder())}>Show folder</button>
          <button type="button" class="btn" disabled={page.Conversations === 0} onclick={() => (confirming = "clear")}>Clear…</button>
        {/if}
      </span>
    </div>
  </div>

  {#if page.Installed && !page.Installing}
    <h3 class="group-title">Storage</h3>
    <div class="group">
      <div class="item">
        <span class="text">
          <span class="name">Remove the model</span>
          <span class="sub">Turns Omni AI off and deletes the model and its program (about 1 GB). Chats stay.</span>
        </span>
        <span class="end">
          {#if confirming === "remove"}
            <button type="button" class="btn danger" onclick={remove}>Delete</button>
            <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
          {:else}
            <button type="button" class="btn" onclick={() => (confirming = "remove")}>Remove…</button>
          {/if}
        </span>
      </div>
    </div>
  {/if}
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
  .item .ask {
    width: auto;
    height: 30px;
  }
  .top {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .top :global(input) {
    flex: 1;
    min-width: 0;
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
  textarea::placeholder {
    color: var(--text-3);
  }
  .item .error {
    margin: 0;
    font-size: 12px;
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
  input[type="radio"] {
    width: 16px;
    height: 16px;
    margin: 0;
    accent-color: var(--chrome);
    cursor: pointer;
  }
  label.item {
    cursor: pointer;
  }
  @media (prefers-reduced-motion: reduce) {
    .fill {
      transition: none;
    }
  }
</style>
