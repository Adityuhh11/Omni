<script lang="ts">
  import { combo, isMac } from "./lib/platform";
  import { onMount, tick } from "svelte";
  import { Browser, Window } from "@wailsio/runtime";
  import { Service as Notes } from "../bindings/omni/internal/notes";
  import type { Note } from "../bindings/omni/internal/notes";
  import "./lib/settings/forms.css";
  import { when } from "./lib/format";
  import Card from "./lib/Card.svelte";
  import Editor from "./lib/notes/Editor.svelte";
  import RoundButton from "./lib/RoundButton.svelte";

  const SAVE_AFTER_MS = 400;
  const PIN_KEY = "omni-notes-on-top";

  let list = $state<Note[]>([]);
  let query = $state("");
  let loaded = $state(false);
  // The note in the editor; id is "" until its first save. `key` is a new editor (and a new undo history) for each note opened.
  let editing = $state<{ id: string; text: string; key: number } | null>(null);
  let opened = 0;
  let status = $state("");
  let onTop = $state(true);
  let confirmDelete = $state(false);
  let problem = $state("");
  let editor = $state<{ focus: () => void } | undefined>();
  let search = $state<HTMLInputElement | undefined>();
  let timer: ReturnType<typeof setTimeout> | undefined;
  let saving: Promise<void> = Promise.resolve(); // saves run one after another, so a slow one never lands after a newer one
  let saveFailed = false; // the last save did not work: the editor must not be left, or what was typed would be lost

  const words = $derived(editing ? editing.text.trim().split(/\s+/).filter(Boolean).length : 0);

  /** Runs a call to the program; if it fails the reason is shown and false comes back. */
  async function show(action: () => Promise<unknown>): Promise<boolean> {
    try {
      problem = "";
      await action();
      return true;
    } catch (error) {
      problem = String((error as Error)?.message ?? error);
      return false;
    }
  }

  let listAsked = 0; // answers can arrive out of order; only the newest question's answer is shown
  async function loadList() {
    const mine = ++listAsked;
    await show(async () => {
      const found = (await Notes.List(query)) ?? [];
      if (mine !== listAsked) return;
      list = found;
      loaded = true;
    });
  }

  function typed(text: string) {
    if (!editing) return;
    editing.text = text;
    status = "";
    clearTimeout(timer);
    timer = setTimeout(flush, SAVE_AFTER_MS);
  }

  /** Saves what is in the editor now, after any save already under way. True if everything is saved. */
  async function flush(): Promise<boolean> {
    clearTimeout(timer);
    const note = editing;
    if (note) {
      const text = note.text;
      saving = saving.then(async () => {
        saveFailed = !(await show(async () => {
          if (note.id === "" && text.trim() === "") return; // nothing typed yet: no note is made
          const saved = await Notes.Save(note.id, text);
          if (note.id === "" && saved.ID) note.id = saved.ID;
          if (editing === note) status = "Saved";
        }));
      });
      await saving;
      if (!saveFailed) loadList(); // its title and preview in the list, and the note first
    }
    await saving;
    return !saveFailed;
  }

  /** Leaves the note in the editor: saves it, and does not keep one that was emptied. False if it could not be saved (it stays open). */
  async function settle(): Promise<boolean> {
    if (!(await flush())) return false;
    const note = editing;
    if (note?.id && note.text.trim() === "") await show(() => Notes.Delete(note.id));
    return true;
  }

  async function begin(next: { id: string; text: string }) {
    editing = { ...next, key: ++opened };
    confirmDelete = false;
    status = "";
    await tick();
    editor?.focus();
  }

  async function open(id: string) {
    if (editing?.id === id) return editor?.focus();
    if (!(await settle())) return;
    await show(async () => begin({ id, text: (await Notes.Get(id)).Text }));
    await loadList();
  }

  async function create(text = "") {
    if (!(await settle())) return;
    await begin({ id: "", text });
    await loadList();
  }

  async function close() {
    if (!(await settle())) return; // the error is on screen and the note is still in the editor
    editing = null;
    confirmDelete = false;
    await loadList();
    await tick();
    search?.focus();
  }

  async function remove() {
    const note = editing;
    clearTimeout(timer);
    editing = null;
    confirmDelete = false;
    await saving; // a first save still under way gives the note its id; deleting before it lands would leave the note behind
    if (note?.id) await show(() => Notes.Delete(note.id));
    await loadList();
  }

  async function togglePin() {
    onTop = !onTop;
    Notes.SetOnTop(onTop);
    try {
      localStorage.setItem(PIN_KEY, String(onTop));
    } catch {
      // a window without storage forgets the choice; it still works
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.ctrlKey && event.key.toLowerCase() === "n") {
      event.preventDefault();
      create();
    } else if (event.key === "Escape") {
      if (confirmDelete) confirmDelete = false;
      else if (query) {
        query = "";
        loadList();
      } else if (editing) close();
    }
  }

  // The address says what the window is for: #notes (the list), #notes/new, or #notes/<id>.
  async function follow() {
    const want = location.hash.split("/")[1] ?? "";
    if (want === "new") await create();
    else if (want !== "") await open(want);
    else if (editing) await close();
    history.replaceState(null, "", "#notes"); // handled: the next request for the same note is a new address, and is heard
  }

  onMount(() => {
    document.documentElement.dataset.surface = "card";
    try {
      onTop = localStorage.getItem(PIN_KEY) !== "false";
    } catch {
      onTop = true;
    }
    Notes.SetOnTop(onTop);
    loadList().then(follow);
    const flushNow = () => void flush();
    window.addEventListener("hashchange", follow);
    window.addEventListener("blur", flushNow);
    window.addEventListener("pagehide", flushNow); // the window is closing: save what is waiting for the delay
    document.addEventListener("visibilitychange", flushNow);
    return () => {
      window.removeEventListener("hashchange", follow);
      window.removeEventListener("blur", flushNow);
      window.removeEventListener("pagehide", flushNow);
      document.removeEventListener("visibilitychange", flushNow);
    };
  });
</script>

<svelte:window onkeydown={onKeydown} />

<Card>
<main>
  <aside aria-label="Notes">
    <header>
      <input
        bind:this={search}
        bind:value={query}
        oninput={loadList}
        type="text"
        placeholder="Search notes"
        aria-label="Search notes"
        spellcheck="false"
        autocomplete="off"
      />
      <RoundButton icon="plus" look="solid" size={30} label="New note" title={`New note (${combo("Ctrl+N")})`} onclick={() => create()} />
    </header>
    {#if list.length > 0}
      <ul class="notes">
        {#each list as note (note.ID)}
          <li>
            <button type="button" class:on={editing?.id === note.ID} aria-current={editing?.id === note.ID} onclick={() => open(note.ID)}>
              <span class="title">{note.Title}</span>
              <span class="when">{when(note.Modified)}</span>
              {#if note.Preview}<span class="preview">{note.Preview}</span>{/if}
            </button>
          </li>
        {/each}
      </ul>
    {:else if loaded}
      <p class="none">{query ? `No note has “${query}” in it.` : "No notes yet."}</p>
    {/if}
  </aside>

  <section aria-label="Note">
    <header>
      <span class="state" aria-live="polite">{editing ? status || (editing.id === "" ? "New note" : "") : ""}</span>
      <div class="tools">
        {#if editing}
          {#if confirmDelete}
            <button type="button" class="btn danger" onclick={remove}>Delete</button>
            <button type="button" class="btn quiet" onclick={() => (confirmDelete = false)}>Keep</button>
          {:else}
            <button type="button" class="btn quiet" onclick={() => (confirmDelete = true)} title="Delete this note">Delete</button>
          {/if}
        {/if}
        <button type="button" class="btn quiet" class:on={onTop} aria-pressed={onTop} onclick={togglePin} title="Keep above other windows">On top</button>
        <RoundButton icon="close" label="Close" onclick={() => Window.Close()} />
      </div>
    </header>
    {#if problem}<p class="error pad">{problem}</p>{/if}
    {#if editing}
      {#key editing.key}
        <Editor
          bind:this={editor}
          text={editing.text}
          placeholder="Start writing. # heading, **bold**, - [ ] task, `code`"
          onchange={typed}
          openLink={(href) => Browser.OpenURL(href)}
        />
      {/key}
      <footer>
        <span>{words} {words === 1 ? "word" : "words"}</span>
        <span>{isMac ? "⌘" : "Ctrl"}-click a link to open it</span>
      </footer>
    {:else}
      <div class="empty">
        <p>{loaded && list.length === 0 && !query ? "Nothing written yet." : "Pick a note, or start a new one."}</p>
        <button type="button" class="btn primary" onclick={() => create()}>New note <kbd>{combo("Ctrl+N")}</kbd></button>
      </div>
    {/if}
  </section>
</main>
</Card>

<style>
  main {
    display: grid;
    grid-template-columns: minmax(200px, 270px) 1fr;
    flex: 1;
    min-height: 0;
    max-width: none;
    color: var(--text);
  }
  aside,
  section {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }
  aside {
    border-right: 1px solid var(--line);
  }
  header {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 56px;
    box-sizing: border-box;
    padding: 12px 14px 10px 16px;
    border-bottom: 1px solid var(--line);
    --wails-draggable: drag; /* the window is moved by holding a header anywhere but on a control */
  }
  header :global(button),
  header input {
    --wails-draggable: no-drag;
  }
  header input {
    flex: 1;
    min-width: 0;
  }
  .tools {
    display: flex;
    gap: 4px;
    margin-left: auto;
  }
  .state {
    color: var(--text-3);
    font-size: 12px;
  }
  :global(main .btn.quiet.on) {
    color: var(--text);
    background: var(--lift);
  }
  footer {
    display: flex;
    justify-content: space-between;
    padding: 6px 16px;
    border-top: 1px solid var(--line);
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
  .notes {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 8px;
    overflow-y: auto;
    list-style: none;
  }
  .notes button {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 2px 10px;
    width: 100%;
    padding: 9px 12px;
    border: 0;
    border-radius: 12px;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    transition: background-color 0.12s;
  }
  .notes button:hover {
    background: var(--tile);
  }
  .notes button.on {
    background: var(--lift);
    box-shadow: inset 2px 0 0 var(--chrome);
  }
  .notes button:active {
    background: var(--lift);
  }
  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: 560 13.5px/1.35 var(--sans);
  }
  .when {
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
    align-self: center;
  }
  .preview {
    grid-column: 1 / -1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-2);
    font-size: 12.5px;
  }
  .none {
    margin: 0;
    padding: 28px 20px;
    color: var(--text-3);
    font-size: 13px;
    text-align: center;
  }
  .pad {
    padding: 8px 16px 0;
  }
  .empty {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    color: var(--text-3);
    font-size: 13px;
  }
  .empty p {
    margin: 0;
  }
  kbd {
    margin-left: 6px;
    opacity: 0.7;
    font: 500 11px var(--mono);
  }
</style>
