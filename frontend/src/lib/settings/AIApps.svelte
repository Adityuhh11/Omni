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

{#if page}
  <h3 class="group-title">MCP server</h3>
  <div class="group">
    <label class="item">
      <span class="text">
        <span class="name">Let AI apps on this computer reach Omni</span>
        <span class="sub">Claude Desktop, Cursor, VS Code and others. What an app reads goes wherever that app sends it.</span>
      </span>
      <input type="checkbox" class="switch" checked={page.Enabled} onchange={(e) => run(() => MCP.SetEnabled(e.currentTarget.checked))} />
    </label>
  </div>
  {#if error}<p class="error">{error}</p>{/if}

  <h3 class="group-title">What they may read</h3>
  <div class="group">
    {#each page.Groups ?? [] as group (group.Group)}
      <div class="item stack">
        <label class="line">
          <span class="name">{group.Title}</span>
          <input type="checkbox" class="switch" checked={group.On} disabled={!page.Enabled} onchange={(e) => run(() => MCP.SetTool(group.Group, e.currentTarget.checked))} />
        </label>
        {#if (group.Tools ?? []).length}
          <div class="tools">
            {#each group.Tools ?? [] as tool (tool.Name)}
              <details class="tool">
                <summary><span class="mono">{tool.Name}</span></summary>
                <p>{tool.Description}</p>
              </details>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  </div>
  <p class="group-note">Only tools that read are offered; tools that change something stay in Omni AI, where it can ask you first. Separate from Omni AI's own tools.</p>

  {#if page.Enabled}
    <h3 class="group-title">Connect an app</h3>
    <div class="group">
      {#each page.Apps ?? [] as app (app.ID)}
        <div class="item">
          <span class="text">
            <span class="name">{app.Name}</span>
            <span class="sub">
              {#if app.State === "on"}Connected
              {:else if app.State === "other"}Set up for an Omni at another place
              {:else if app.State === "problem"}{app.Problem}
              {:else}Not connected{/if}
            </span>
          </span>
          {#if app.State === "on"}
            <button type="button" class="btn" onclick={() => run(() => MCP.Disconnect(app.ID))}>Disconnect</button>
          {:else if app.State !== "problem"}
            <button type="button" class="btn" onclick={() => run(() => MCP.Connect(app.ID))}>{app.State === "other" ? "Update" : "Connect"}</button>
          {/if}
        </div>
      {:else}
        <p class="item sub">No supported app found (Claude Desktop, Cursor, VS Code, Windsurf).</p>
      {/each}
      <div class="item stack">
        <span class="line">
          <span class="name">Any other app</span>
          <button type="button" class="btn" onclick={copyConfig}>{copied ? "Copied" : "Copy"}</button>
        </span>
        <pre class="mono config">{page.Config}</pre>
        <span class="sub">Claude Code: <span class="mono">claude mcp add omni -- "{page.Self}" mcp</span></span>
      </div>
    </div>
    {#if (page.Apps ?? []).length}
      <p class="group-note">Connect keeps the app's other servers, and the first time the old file as <span class="mono">.omni-backup</span>. Restart the app after.</p>
    {/if}
  {/if}
{/if}

<style>
  .line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    cursor: pointer;
  }
  .tools {
    display: flex;
    flex-direction: column;
    margin-top: -4px;
  }
  .tool {
    color: var(--text-2);
    font: 12.5px/1.45 var(--sans);
  }
  .tool summary {
    width: fit-content;
    padding: 2px 0;
    color: var(--text-3);
    cursor: pointer;
  }
  .tool summary:hover .mono {
    color: var(--text);
  }
  .tool p {
    margin: 2px 0 6px;
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
  label.item {
    cursor: pointer;
  }
</style>
