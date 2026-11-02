<script lang="ts">
  import { onMount } from "svelte";
  import { Clipboard } from "@wailsio/runtime";
  import { MCP } from "../../../bindings/omni/internal/settingsui";
  import type { MCPView } from "../../../bindings/omni/internal/settingsui";

  let page = $state<MCPView | null>(null);
  let error = $state("");
  let copied = $state(false);

  const refresh = async () => {
    page = await MCP.View();
  };

  // Scripts and tools come and go, so the list is asked for again now and then.
  onMount(() => {
    refresh();
    const timer = setInterval(refresh, 2000);
    return () => clearInterval(timer);
  });

  /** Runs a change, shows why it was refused, and shows the page as it is now. */
  async function run(action: () => Promise<string | void>) {
    error = (await action()) ?? "";
    await refresh();
  }

  async function copyConfig() {
    if (!page) return;
    await Clipboard.SetText(page.Config);
    copied = true;
    setTimeout(() => (copied = false), 1500);
  }
</script>

<h2>AI apps</h2>
<p class="lede">
  Let Claude Desktop, Cursor, VS Code and other apps that speak MCP search your files, notes and more through Omni. They ask, Omni answers from
  what it keeps on this computer. Omni must be running.
</p>

{#if page}
  <section class="block">
    <div class="field">
      <label class="check"><input type="checkbox" checked={page.Enabled} onchange={(e) => run(() => MCP.SetEnabled(e.currentTarget.checked))} /> Let AI apps on this computer reach Omni</label>
      <p class="help">
        Nothing is shared until you switch on a group below. What an app reads goes wherever that app sends it, which may be a server on the internet.
        This is separate from Omni AI: a group on there is not on here.
      </p>
    </div>
    {#if error}<p class="error">{error}</p>{/if}
  </section>

  <section class="block">
    <h3>What they may use</h3>
    {#each page.Groups ?? [] as group (group.Group)}
      <div class="field">
        <label class="check"><input type="checkbox" checked={group.On} disabled={!page.Enabled} onchange={(e) => run(() => MCP.SetTool(group.Group, e.currentTarget.checked))} /> {group.Title}</label>
        {#each group.Tools ?? [] as tool (tool.Name)}
          <details class="tool">
            <summary><span class="mono">{tool.Name}</span></summary>
            <p>{tool.Description}</p>
          </details>
        {/each}
      </div>
    {/each}
    <p class="help">Click a tool to see what the other app is told about it. Only tools that read are offered. Tools that change something, such as adding a note or running a script, stay in Omni, where it can ask you first.</p>
  </section>

  {#if page.Enabled}
    <section class="block">
      <h3>Connect an app</h3>
      {#each page.Apps ?? [] as app (app.ID)}
        <div class="app">
          <div class="app-text">
            <span class="app-name">{app.Name}</span>
            <span class="help">
              {#if app.State === "on"}Connected.
              {:else if app.State === "other"}Set up for an Omni at another place.
              {:else if app.State === "problem"}{app.Problem}
              {:else}Not connected.{/if}
            </span>
          </div>
          {#if app.State === "on"}
            <button type="button" class="btn" onclick={() => run(() => MCP.Disconnect(app.ID))}>Disconnect</button>
          {:else if app.State !== "problem"}
            <button type="button" class="btn" onclick={() => run(() => MCP.Connect(app.ID))}>{app.State === "other" ? "Update" : "Connect"}</button>
          {/if}
        </div>
      {/each}
      {#if (page.Apps ?? []).length}
        <p class="help">Adds Omni to that app's MCP settings and keeps what is already there; the first time, the old file is kept beside it as <span class="mono">.omni-backup</span>. Restart the app after.</p>
      {:else}
        <p class="help">No supported app found here (Claude Desktop, Cursor, VS Code, Windsurf). Add this by hand instead.</p>
      {/if}
      <div class="field">
        <span class="label">Any other app: add this to its MCP settings</span>
        <pre class="mono config">{page.Config}</pre>
        <div><button type="button" class="btn" onclick={copyConfig}>{copied ? "Copied" : "Copy"}</button></div>
        <p class="help">Claude Code: <span class="mono">claude mcp add omni -- "{page.Self}" mcp</span></p>
      </div>
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
  .tool {
    margin: 2px 0 0 26px;
    color: var(--text-2);
    font: 12.5px/1.45 var(--sans);
  }
  .tool summary {
    width: fit-content;
    padding: 2px 0;
    cursor: pointer;
  }
  .tool summary:hover .mono {
    color: var(--text);
  }
  .tool p {
    margin: 2px 0 6px;
  }
  .app {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .app-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .app-name {
    font: 560 13px/1.3 var(--sans);
  }
  .config {
    margin: 0;
    padding: 10px 12px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--tile);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }
</style>
