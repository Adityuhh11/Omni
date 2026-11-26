<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Window } from "@wailsio/runtime";
  import { Service as Notes } from "../bindings/omni/internal/notes";
  import type { Note } from "../bindings/omni/internal/notes";
  import "./lib/settings/forms.css";
  import { when } from "./lib/format";
  import { combo } from "./lib/platform";
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

  const runs = $derived(grouped(list));

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
    } catch (error) {
      problem = String((error as Error)?.message ?? error);
    }
  }

  /** The note window takes the note over, and this window has done its job. */
  async function open(id: string) {
    await Notes.OpenNote(id);
    Window.Close();
  }

  async function move(to: number) {
    cursor = to;
    await tick();
    listing?.querySelector(".on")?.scrollIntoView({ block: "nearest" });
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      if (query) {
        query = "";
        cursor = 0;
        load();
      } else Window.Close();
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (list.length > 0) move((cursor + (event.key === "ArrowDown" ? 1 : list.length - 1)) % list.length);
    } else if (event.key === "Enter" && list[cursor]) {
      event.preventDefault();
      open(list[cursor].ID);
    } else if (event.ctrlKey && event.key.toLowerCase() === "n") {
      event.preventDefault();
      open("new");
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
    <RoundButton icon="plus" look="solid" size={30} label="New note" title={`New note (${combo("Ctrl+N")})`} onclick={() => open("new")} />
    <RoundButton icon="close" label="Close" onclick={() => Window.Close()} />
  </header>
  {#if problem}<p class="error pad">{problem}</p>{/if}
  <div class="scroll" bind:this={listing}>
    {#each runs as run (run.heading)}
      <h2>{run.heading}</h2>
      <ul>
        {#each run.notes as note (note.ID)}
          <li>
            <button type="button" class:on={list[cursor]?.ID === note.ID} onmousemove={() => (cursor = list.indexOf(note))} onclick={() => open(note.ID)}>
              <span class="title">{note.Title}</span>
              <span class="when">{when(note.Modified)}</span>
              {#if note.Preview}<span class="preview">{note.Preview}</span>{/if}
            </button>
          </li>
        {/each}
      </ul>
    {/each}
    {#if loaded && list.length === 0}
      <p class="none">{query ? `No note has “${query}” in it.` : "No notes yet. Press + to write the first."}</p>
    {/if}
  </div>
  <footer>{loaded ? `${list.length} ${list.length === 1 ? "note" : "notes"}${query ? " found" : ""}` : ""}</footer>
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
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 3px 12px;
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
  li button:active {
    background: var(--lift-keyed);
  }
  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: 580 14px/1.35 var(--sans);
  }
  .when {
    align-self: center;
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
  .preview {
    display: -webkit-box;
    grid-column: 1 / -1;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.45;
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
    padding: 5px 14px;
    border-top: 1px solid var(--line);
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
</style>
