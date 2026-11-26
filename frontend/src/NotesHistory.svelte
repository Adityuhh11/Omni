<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Window } from "@wailsio/runtime";
  import { Service as Notes } from "../bindings/omni/internal/notes";
  import type { Note } from "../bindings/omni/internal/notes";
  import "./lib/settings/forms.css";
  import { when } from "./lib/format";
  import { combo } from "./lib/platform";
  import { announceGone } from "./lib/notes/gone";
  import { grouped } from "./lib/notes/groups";
  import Card from "./lib/Card.svelte";
  import Icon from "./lib/Icon.svelte";
  import RoundButton from "./lib/RoundButton.svelte";

  let list = $state<Note[]>([]);
  let query = $state("");
  let loaded = $state(false);
  let problem = $state("");
  let cursor = $state(0);
  let search = $state<HTMLInputElement | undefined>();
  let listing = $state<HTMLElement | undefined>();
  // Selecting: a click ticks a note instead of opening it, and the notes ticked can be deleted together.
  let selecting = $state(false);
  let picked = $state(new Set<string>());
  let confirming = $state(false);
  let anchor = -1; // the last note ticked, where a Shift-click's range starts

  const runs = $derived(grouped(list));
  const count = $derived(picked.size);

  let asked = 0; // answers can arrive out of order; only the newest question's answer is shown
  async function load() {
    const mine = ++asked;
    try {
      const found = (await Notes.List(query)) ?? [];
      if (mine !== asked) return;
      list = found;
      problem = "";
      loaded = true;
      cursor = Math.min(cursor, Math.max(0, found.length - 1));
      picked = new Set([...picked].filter((id) => found.some((n) => n.ID === id))); // a note that is gone cannot stay ticked
    } catch (error) {
      problem = String((error as Error)?.message ?? error);
    }
  }

  /** The note window takes the note over, and this window has done its job. */
  async function open(id: string) {
    await Notes.OpenNote(id);
    Window.Close();
  }

  function toggle(index: number, range: boolean) {
    const next = new Set(picked);
    if (range && anchor >= 0) {
      const [from, to] = [Math.min(anchor, index), Math.max(anchor, index)];
      for (let i = from; i <= to; i++) next.add(list[i].ID);
    } else if (next.has(list[index].ID)) next.delete(list[index].ID);
    else next.add(list[index].ID);
    anchor = index;
    picked = next;
    confirming = false;
  }

  function choose(index: number, event?: MouseEvent) {
    cursor = index;
    if (selecting) toggle(index, event?.shiftKey ?? false);
    else open(list[index].ID);
  }

  function startSelecting() {
    selecting = true;
    anchor = -1;
  }

  function stopSelecting() {
    selecting = false;
    confirming = false;
    picked = new Set();
  }

  const selectAll = () => {
    picked = picked.size === list.length ? new Set() : new Set(list.map((n) => n.ID));
    confirming = false;
  };

  async function removePicked() {
    const ids = [...picked];
    confirming = false;
    try {
      for (const id of ids) await Notes.Delete(id);
    } catch (error) {
      problem = String((error as Error)?.message ?? error);
    }
    announceGone(ids);
    stopSelecting();
    await load();
  }

  async function move(to: number) {
    cursor = to;
    await tick();
    listing?.querySelector(".on")?.scrollIntoView({ block: "nearest" });
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      if (confirming) confirming = false;
      else if (selecting) stopSelecting();
      else if (query) {
        query = "";
        cursor = 0;
        load();
      } else Window.Close();
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (list.length > 0) move((cursor + (event.key === "ArrowDown" ? 1 : list.length - 1)) % list.length);
    } else if (event.key === "Enter" && list[cursor]) {
      event.preventDefault();
      choose(cursor);
    } else if (event.ctrlKey && event.key.toLowerCase() === "n") {
      event.preventDefault();
      open("new");
    } else if (event.ctrlKey && event.key.toLowerCase() === "a" && selecting && document.activeElement !== search) {
      event.preventDefault();
      selectAll();
    } else if (event.key === "Delete" && selecting && count > 0 && document.activeElement !== search) {
      confirming = true;
    }
  }

  onMount(() => {
    document.documentElement.dataset.surface = "card";
    load();
    search?.focus();
    window.addEventListener("focus", load); // a note was changed in its own window while this one was behind it
    return () => window.removeEventListener("focus", load);
  });
</script>

<svelte:window onkeydown={onKeydown} />

<Card>
<main>
  <header>
    <label class="search">
      <Icon name="search" size={16} />
      <input bind:this={search} bind:value={query} oninput={() => ((cursor = 0), load())} type="text" placeholder="Search your notes" aria-label="Search notes" spellcheck="false" autocomplete="off" />
    </label>
    {#if list.length > 0 || selecting}
      <button type="button" class="btn quiet" class:on={selecting} aria-pressed={selecting} onclick={() => (selecting ? stopSelecting() : startSelecting())}>
        {selecting ? "Done" : "Select"}
      </button>
    {/if}
    <RoundButton icon="plus" look="solid" size={30} label="New note" title={`New note (${combo("Ctrl+N")})`} onclick={() => open("new")} />
    <RoundButton icon="close" label="Close" onclick={() => Window.Close()} />
  </header>
  {#if problem}<p class="error pad">{problem}</p>{/if}
  <div class="scroll" bind:this={listing}>
    {#each runs as run (run.heading)}
      <h2>{run.heading}</h2>
      <ul>
        {#each run.notes as note (note.ID)}
          {@const index = list.indexOf(note)}
          <li>
            <button
              type="button"
              class:on={cursor === index}
              class:picked={picked.has(note.ID)}
              role={selecting ? "checkbox" : undefined}
              aria-checked={selecting ? picked.has(note.ID) : undefined}
              onmousemove={() => (cursor = index)}
              onclick={(event) => choose(index, event)}
            >
              {#if selecting}<span class="box" aria-hidden="true"></span>{/if}
              <span class="text">
                <span class="title">{note.Title}</span>
                {#if note.Preview}<span class="preview">{note.Preview}</span>{/if}
              </span>
              <span class="when">{when(note.Modified)}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/each}
    {#if loaded && list.length === 0}
      <p class="none">{query ? `No note has “${query}” in it.` : "No notes yet. Press + to write the first."}</p>
    {/if}
  </div>
  <footer>
    {#if selecting}
      {#if confirming}
        <span class="ask">Delete {count} {count === 1 ? "note" : "notes"}? This cannot be undone.</span>
        <span class="acts">
          <button type="button" class="btn danger" onclick={removePicked}>Delete</button>
          <button type="button" class="btn quiet" onclick={() => (confirming = false)}>Keep</button>
        </span>
      {:else}
        <span>{count === 0 ? "Pick the notes to delete" : `${count} selected`}</span>
        <span class="acts">
          <button type="button" class="btn quiet" onclick={selectAll}>{count === list.length && count > 0 ? "None" : "All"}</button>
          <button type="button" class="btn danger" disabled={count === 0} onclick={() => (confirming = true)}>Delete{count > 0 ? ` ${count}` : ""}</button>
        </span>
      {/if}
    {:else}
      <span>{loaded ? `${list.length} ${list.length === 1 ? "note" : "notes"}${query ? " found" : ""}` : ""}</span>
    {/if}
  </footer>
</main>
</Card>

<style>
  main {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    max-width: none;
    color: var(--text);
  }
  header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 12px 10px 14px;
    border-bottom: 1px solid var(--line);
    --wails-draggable: drag; /* the window is moved by holding the header anywhere but on a control */
  }
  header :global(button),
  header input {
    --wails-draggable: no-drag;
  }
  :global(main .btn.quiet.on) {
    color: var(--text);
    background: var(--lift);
  }
  .search {
    display: flex;
    flex: 1;
    align-items: center;
    gap: 10px;
    min-width: 0;
    padding: 0 12px;
    border: 1px solid var(--line);
    border-radius: 999px;
    background: var(--tile);
    color: var(--text-3);
  }
  .search:focus-within {
    border-color: color-mix(in srgb, var(--chrome) 45%, transparent);
    color: var(--chrome);
  }
  .search input[type="text"] {
    flex: 1;
    min-width: 0;
    height: 34px;
    padding: 0;
    border: 0;
    outline: 0;
    background: transparent;
    font: 450 13.5px/1 var(--sans);
  }
  .scroll {
    flex: 1;
    min-height: 0;
    padding: 4px 8px 10px;
    overflow-y: auto;
  }
  h2 {
    margin: 14px 12px 4px;
    color: var(--text-3);
    font: 600 11px/1 var(--sans);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li button {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    width: 100%;
    padding: 10px 12px;
    border: 0;
    border-radius: 12px;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    transition: background-color 0.12s;
  }
  li button.on {
    background: var(--lift);
    box-shadow: inset 2px 0 0 var(--chrome);
  }
  li button.picked {
    background: color-mix(in srgb, var(--chrome) 14%, transparent);
  }
  li button:active {
    background: var(--lift-keyed);
  }
  .text {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: 580 14px/1.35 var(--sans);
  }
  .when {
    flex: none;
    margin-top: 2px;
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
  .preview {
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.45;
  }
  /* The tick box of a note while selecting: a ring that fills with the accent and shows a check. */
  .box {
    position: relative;
    flex: none;
    width: 18px;
    height: 18px;
    margin-top: 1px;
    border: 1.5px solid var(--text-3);
    border-radius: 50%;
    transition: background-color 0.12s, border-color 0.12s;
  }
  .picked .box {
    border-color: var(--chrome);
    background: var(--chrome);
  }
  .picked .box::after {
    content: "";
    position: absolute;
    left: 5px;
    top: 2px;
    width: 4px;
    height: 8px;
    border: solid var(--on-chrome);
    border-width: 0 2px 2px 0;
    transform: rotate(45deg);
  }
  .none {
    margin: 0;
    padding: 36px 20px;
    color: var(--text-3);
    font-size: 13px;
    text-align: center;
  }
  .pad {
    padding: 8px 14px 0;
  }
  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    min-height: 40px;
    box-sizing: border-box;
    padding: 5px 12px 5px 14px;
    border-top: 1px solid var(--line);
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
  .ask {
    color: var(--text);
  }
  .acts {
    display: flex;
    gap: 4px;
  }
</style>
