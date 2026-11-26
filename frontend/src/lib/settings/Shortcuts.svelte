<script lang="ts">
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { View } from "../../../bindings/omni/internal/settingsui";
  import { Service as Palette } from "../../../bindings/omni/internal/palette";
  import type { Item } from "../../../bindings/omni/internal/palette";
  import HotkeyField from "./HotkeyField.svelte";

  let { view }: { view: View } = $props();

  type Row = { id: number; keys: string; target: string; title: string };
  let nextId = 0;
  // The list starts from the shortcuts as they were when the page opened and then owns its rows. A row is saved once it has a hotkey.
  // svelte-ignore state_referenced_locally
  let rows = $state<Row[]>((view.Shortcuts ?? []).map((s) => ({ id: nextId++, keys: s.Keys, target: s.Target, title: s.Title || s.Target })));
  const failing = (keys: string) => view.ShortcutErrors?.[keys] ?? "";
  let error = $state("");

  async function save(): Promise<string> {
    const list = rows.filter((r) => r.keys !== "").map((r) => ({ Keys: r.keys, Target: r.target, Title: r.title }));
    return (error = await Settings.SaveShortcuts(list));
  }

  async function setKeys(row: Row, keys: string): Promise<string> {
    const before = row.keys;
    row.keys = keys;
    const why = await save();
    if (why) row.keys = before; // refused: the row keeps the combination it had
    return why;
  }

  async function remove(row: Row) {
    rows = rows.filter((r) => r !== row);
    await save();
  }

  /** What a row runs, in a word: the id's first part. */
  function kind(target: string): string {
    if (target.startsWith("cmd:settings")) return "Settings page";
    const names: Record<string, string> = { app: "App", script: "Script", timer: "Timer", cal: "Calendar", quicklink: "Quicklink" };
    return names[target.split(":")[0]] ?? "Command";
  }

  // The picker asks the bar for rows, so it offers what the bar would run: apps, scripts, settings pages, the timer.
  const RUNNABLE = new Set(["app", "script", "cmd", "timer", "calendar", "system", "quicklink"]);
  let picking = $state(false);
  let text = $state("");
  let found = $state<Item[]>([]);
  let asked = 0;

  async function search() {
    const mine = ++asked;
    const answer = text.trim() === "" ? null : await Palette.Query(text, "all");
    if (mine !== asked) return;
    found = (answer?.Items ?? []).filter((i) => RUNNABLE.has(i.Kind) && !i.Fill).slice(0, 8);
  }

  function pick(item: Item) {
    rows.push({ id: nextId++, keys: "", target: item.ID, title: item.Title });
    picking = false;
    text = "";
    found = [];
  }
</script>

<div class="head-row">
  <h2>Hotkeys</h2>
</div>

<h3 class="group-title">Open an app, run a script or a command from anywhere</h3>
<div class="group">
  {#each rows as row (row.id)}
    <div class="item stack">
      <div class="top">
        <span class="text">
          <span class="name" id="shortcut-{row.id}">{row.title}</span>
          <span class="sub">{kind(row.target)}</span>
        </span>
        <button type="button" class="btn quiet" aria-label="Remove the hotkey for {row.title}" onclick={() => remove(row)}>✕</button>
      </div>
      <HotkeyField value={row.keys} labelId="shortcut-{row.id}" onsave={(keys) => setKeys(row, keys)} />
      {#if row.keys !== "" && failing(row.keys)}<p class="error">{failing(row.keys)}</p>{/if}
    </div>
  {:else}
    {#if !picking}<p class="item sub">No hotkeys yet.</p>{/if}
  {/each}
  {#if picking}
    <div class="item stack">
      <!-- svelte-ignore a11y_autofocus -->
      <input type="text" aria-label="Search for something to run" placeholder="Search for an app, a script or a command" bind:value={text} oninput={search} autofocus spellcheck="false" />
      {#each found as item (item.ID)}
        <button type="button" class="pick" onclick={() => pick(item)}>
          <span class="name">{item.Title}</span>
          <span class="sub">{item.Subtitle || kind(item.ID)}</span>
        </button>
      {:else}
        {#if text.trim() !== ""}<p class="sub">Nothing found. Apps, scripts, settings pages and the timer can have a hotkey.</p>{/if}
      {/each}
      <span class="end"><button type="button" class="btn quiet" onclick={() => ((picking = false), (text = ""), (found = []))}>Cancel</button></span>
    </div>
  {:else}
    <div class="item">
      <button type="button" class="btn" onclick={() => (picking = true)}>Add hotkey</button>
      {#if rows.some((r) => r.keys === "")}<span class="sub">Press Set on the new row to choose its keys. It is saved once it has them.</span>{/if}
    </div>
  {/if}
</div>
{#if error}<p class="error">{error}</p>{/if}

<style>
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .pick {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 100%;
    padding: 8px 10px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--text);
    text-align: left;
    cursor: pointer;
  }
  .pick:hover,
  .pick:focus-visible {
    background: var(--lift);
    outline: none;
  }
</style>
