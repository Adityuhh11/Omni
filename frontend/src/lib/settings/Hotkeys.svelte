<script lang="ts">
  import { onMount } from "svelte";
  import { OmniAI, Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { ClipboardView, NotesHotkeyView, OmniAIView, View } from "../../../bindings/omni/internal/settingsui";
  import { Service as Palette } from "../../../bindings/omni/internal/palette";
  import type { Item } from "../../../bindings/omni/internal/palette";
  import { isMac } from "../platform";
  import Icon from "../Icon.svelte";
  import RoundButton from "../RoundButton.svelte";
  import HotkeyField from "./HotkeyField.svelte";

  let { view }: { view: View } = $props();

  // Omni's own hotkeys live on the pages of the services they belong to; they are shown here together and saved there.
  let clipboard = $state<ClipboardView | null>(null);
  let chat = $state<OmniAIView | null>(null);
  let notes = $state<NotesHotkeyView | null>(null);

  async function loadHotkeys() {
    [clipboard, chat, notes] = await Promise.all([Settings.Clipboard(), OmniAI.View(), Settings.NotesHotkey()]);
  }
  onMount(loadHotkeys);

  // The field shows its own refusal ("that hotkey is taken"). The browser and search address are saved with the bar's hotkey, so they are
  // read as they are now, not as they were when the window opened: General may have changed them since.
  async function saveBarKey(accelerator: string): Promise<string> {
    const now = await Settings.View();
    return afterKey(await Settings.SaveGeneral(accelerator, now.Browser, now.SearchURL));
  }
  // The page reloads the hotkeys after a save, so an error from startup goes away once the key works.
  async function afterKey(refused: string): Promise<string> {
    if (refused === "") await loadHotkeys();
    return refused;
  }
  const saveClipboardKey = async (accelerator: string) => {
    const now = await Settings.Clipboard();
    return afterKey(await Settings.SaveClipboard(now.Enabled, now.Ignore ?? [], accelerator));
  };
  const saveChatKey = async (accelerator: string) => afterKey(await OmniAI.SetHotkey(accelerator));
  const saveNotesKey = async (accelerator: string) => afterKey(await Settings.SetNotesHotkey(accelerator));

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
  let cursor = $state(0);
  let asked = 0;

  async function search() {
    const mine = ++asked;
    const answer = text.trim() === "" ? null : await Palette.Query(text, "all");
    if (mine !== asked) return;
    found = (answer?.Items ?? []).filter((i) => RUNNABLE.has(i.Kind) && !i.Fill).slice(0, 8);
    cursor = 0;
  }

  function cancel() {
    picking = false;
    text = "";
    found = [];
  }

  function pickerKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation(); // Escape closes the picker, not Settings
      cancel();
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (found.length > 0) cursor = (cursor + (event.key === "ArrowDown" ? 1 : found.length - 1)) % found.length;
    } else if (event.key === "Enter" && found[cursor]) {
      event.preventDefault();
      pick(found[cursor]);
    }
  }

  function pick(item: Item) {
    rows.push({ id: nextId++, keys: "", target: item.ID, title: item.Title });
    cancel();
  }
</script>

<div class="head-row">
  <h2>Hotkeys</h2>
</div>

<h3 class="group-title">Open Omni from anywhere</h3>
<div class="group">
  <div class="item">
    <span class="name" id="hotkey-label">Search bar</span>
    <HotkeyField labelId="hotkey-label" value={view.Hotkey} defaultValue={view.DefaultHotkey} onsave={saveBarKey} />
  </div>
  {#if clipboard && chat && notes}
    <div class="item">
      <span class="name" id="clip-hotkey-label">Clipboard history</span>
      <HotkeyField labelId="clip-hotkey-label" value={clipboard.Hotkey} clearable onsave={saveClipboardKey} />
    </div>
    {#if clipboard.HotkeyError}<p class="item error">{clipboard.HotkeyError}</p>{/if}
    <div class="item">
      <span class="name" id="ai-hotkey-label">Omni AI</span>
      <HotkeyField labelId="ai-hotkey-label" value={chat.Hotkey} clearable onsave={saveChatKey} />
    </div>
    {#if chat.HotkeyError}<p class="item error">{chat.HotkeyError}</p>{/if}
    <div class="item">
      <span class="name" id="notes-hotkey-label">Notes</span>
      <HotkeyField labelId="notes-hotkey-label" value={notes.Hotkey} clearable onsave={saveNotesKey} />
    </div>
    {#if notes.Error}<p class="item error">{notes.Error}</p>{/if}
  {/if}
</div>
{#if !isMac}<p class="group-note">Windows-key combinations go in config.toml.</p>{/if}

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
      <div class="search">
        <Icon name="search" size={16} />
        <!-- svelte-ignore a11y_autofocus -->
        <input
          type="text"
          role="combobox"
          aria-label="Search for something to run"
          aria-expanded={found.length > 0}
          aria-controls="pick-list"
          placeholder="Search apps, scripts, settings pages…"
          bind:value={text}
          oninput={search}
          onkeydown={pickerKey}
          autofocus
          spellcheck="false"
        />
        <button type="button" class="btn quiet" onclick={cancel}>Cancel</button>
      </div>
      {#if found.length > 0}
        <div class="picks" id="pick-list" role="listbox" aria-label="Results">
          {#each found as item, i (item.ID)}
            <button type="button" role="option" aria-selected={i === cursor} class="pick" class:on={i === cursor} onmousemove={() => (cursor = i)} onclick={() => pick(item)}>
              {#if item.Icon}<img src={item.Icon} alt="" draggable="false" />{/if}
              <span class="words">
                <span class="name">{item.Title}</span>
                <span class="sub">{item.Subtitle || kind(item.ID)}</span>
              </span>
              <span class="kind">{kind(item.ID)}</span>
            </button>
          {/each}
        </div>
      {:else if text.trim() !== ""}
        <p class="sub">Nothing found. Apps, scripts, settings pages and the timer can have a hotkey.</p>
      {/if}
    </div>
  {:else}
    <div class="item add">
      <RoundButton icon="plus" look="solid" label="Add a hotkey" onclick={() => (picking = true)} />
      <span class="sub">
        {rows.some((r) => r.keys === "") ? "Press Set on the new row to choose its keys. It is saved once it has them." : "Add a hotkey for an app, a script or a command"}
      </span>
    </div>
  {/if}
</div>
{#if error}<p class="error">{error}</p>{/if}

<style>
  .item.error {
    min-height: 0;
    padding-top: 0;
    border-top: 0;
    font-size: 12px;
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .add {
    justify-content: flex-start;
    gap: 12px;
  }
  .item.sub {
    color: var(--text-3);
    font-size: 12.5px;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 6px 0 14px;
    border: 1px solid var(--line);
    border-radius: 12px;
    background: var(--tile);
    color: var(--text-3);
  }
  .search:focus-within {
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
    color: var(--chrome);
  }
  .search input[type="text"] {
    flex: 1;
    height: 40px;
    padding: 0;
    border: 0;
    outline: 0;
    background: transparent;
    font: 450 14px/1 var(--sans);
  }
  .picks {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .pick {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 7px 10px;
    border: 0;
    border-radius: 10px;
    background: transparent;
    color: var(--text);
    text-align: left;
    cursor: pointer;
  }
  .pick.on,
  .pick:focus-visible {
    background: var(--lift);
    outline: none;
  }
  .pick img {
    flex: none;
    width: 26px;
    height: 26px;
  }
  .words {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .kind {
    flex: none;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--tile);
    color: var(--text-3);
    font: 500 11px/1.4 var(--mono);
  }
</style>
