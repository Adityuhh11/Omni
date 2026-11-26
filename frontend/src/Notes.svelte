<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Browser, Window } from "@wailsio/runtime";
  import { Service as Notes } from "../bindings/omni/internal/notes";
  import "./lib/settings/forms.css";
  import { combo } from "./lib/platform";
  import Card from "./lib/Card.svelte";
  import Editor from "./lib/notes/Editor.svelte";
  import RoundButton from "./lib/RoundButton.svelte";
  import { onGone } from "./lib/notes/gone";

  const SAVE_AFTER_MS = 400;
  const PIN_KEY = "omni-notes-on-top";
  const LAST_KEY = "omni-notes-last"; // the note a plain opening of the window goes back to

  // The note in the editor; id is "" until its first save. `key` is a new editor (and a new undo history) for each note opened.
  let editing = $state<{ id: string; text: string; key: number } | null>(null);
  let opened = 0;
  let status = $state("");
  let onTop = $state(true);
  let confirmDelete = $state(false);
  let problem = $state("");
  let editor = $state<{ focus: () => void } | undefined>();
  let timer: ReturnType<typeof setTimeout> | undefined;
  let saving: Promise<void> = Promise.resolve(); // saves run one after another, so a slow one never lands after a newer one
  let saveFailed = false; // the last save did not work: the note must not be left, or what was typed would be lost

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

  const remember = (id: string) => {
    try {
      if (id) localStorage.setItem(LAST_KEY, id);
      else localStorage.removeItem(LAST_KEY);
    } catch {
      // a window without storage opens on a new note each time; it still works
    }
  };

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
          if (note.id === "" && saved.ID) {
            note.id = saved.ID;
            remember(saved.ID);
          }
          if (editing === note) status = "Saved";
        }));
      });
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
    remember(next.id);
    await tick();
    editor?.focus();
  }

  async function open(id: string) {
    if (editing?.id === id) return editor?.focus();
    if (!(await settle())) return;
    if (!(await show(async () => begin({ id, text: (await Notes.Get(id)).Text })))) await begin({ id: "", text: "" });
  }

  async function create() {
    if (!(await settle())) return;
    await begin({ id: "", text: "" });
  }

  async function remove() {
    const note = editing;
    clearTimeout(timer);
    confirmDelete = false;
    await saving; // a first save still under way gives the note its id; deleting before it lands would leave the note behind
    if (note?.id) await show(() => Notes.Delete(note.id));
    await begin({ id: "", text: "" });
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
    } else if (event.key === "Escape" && confirmDelete) confirmDelete = false;
  }

  // The address says what the window is for: #notes (the note it had, or a new one), #notes/new, or #notes/<id>.
  async function follow() {
    const want = location.hash.split("/")[1] ?? "";
    if (want === "new") await create();
    else if (want !== "") await open(want);
    else if (!editing) {
      let last = "";
      try {
        last = localStorage.getItem(LAST_KEY) ?? "";
      } catch {
        // no storage: a new note
      }
      try {
        if (!last) throw new Error("no note to go back to");
        await begin({ id: last, text: (await Notes.Get(last)).Text });
      } catch {
        remember(""); // it was deleted, or there was none: a new note, and no complaint
        await begin({ id: "", text: "" });
      }
    }
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
    follow();
    const flushNow = () => void flush();
    // The window is closing: save what is waiting for the delay, and do not keep a note that was emptied.
    const leaving = () => void settle();
    // The history window deleted notes: one of them may be this one, and saving it again would bring it back.
    const stopGone = onGone((ids) => {
      if (!editing || !ids.includes(editing.id)) return;
      clearTimeout(timer);
      begin({ id: "", text: "" });
    });
    window.addEventListener("hashchange", follow);
    window.addEventListener("blur", flushNow);
    window.addEventListener("pagehide", leaving);
    document.addEventListener("visibilitychange", flushNow);
    return () => {
      stopGone();
      window.removeEventListener("hashchange", follow);
      window.removeEventListener("blur", flushNow);
      window.removeEventListener("pagehide", leaving);
      document.removeEventListener("visibilitychange", flushNow);
    };
  });
</script>

<svelte:window onkeydown={onKeydown} />

<Card>
<main>
  <header>
    <RoundButton icon="history" label="All notes" title="All notes" onclick={() => Notes.OpenHistory()} />
    <RoundButton icon="plus" look="solid" size={28} label="New note" title={`New note (${combo("Ctrl+N")})`} onclick={create} />
    <span class="state" aria-live="polite">{editing ? status || (editing.id === "" ? "New note" : "") : ""}</span>
    <div class="tools">
      {#if confirmDelete}
        <button type="button" class="btn danger" onclick={remove}>Delete</button>
        <button type="button" class="btn quiet" onclick={() => (confirmDelete = false)}>Keep</button>
      {:else}
        <button type="button" class="btn quiet" onclick={() => (confirmDelete = true)} title="Delete this note">Delete</button>
      {/if}
      <button type="button" class="btn quiet" class:on={onTop} aria-pressed={onTop} onclick={togglePin} title="Keep above other windows">Pin</button>
      <RoundButton icon="close" label="Close" onclick={() => Window.Close()} />
    </div>
  </header>
  {#if problem}<p class="error pad">{problem}</p>{/if}
  {#if editing}
    {#key editing.key}
      <Editor
        bind:this={editor}
        text={editing.text}
        placeholder="Start writing. # heading, **bold**, - [ ] task"
        onchange={typed}
        openLink={(href) => Browser.OpenURL(href)}
      />
    {/key}
  {/if}
  <footer>{words} {words === 1 ? "word" : "words"}</footer>
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
    gap: 6px;
    min-height: 50px;
    box-sizing: border-box;
    padding: 10px 12px 8px 14px;
    border-bottom: 1px solid var(--line);
    --wails-draggable: drag; /* the window is moved by holding the header anywhere but on a control */
  }
  header :global(button) {
    --wails-draggable: no-drag;
  }
  .tools {
    display: flex;
    gap: 2px;
    margin-left: auto;
  }
  .state {
    margin-left: 6px;
    overflow: hidden;
    color: var(--text-3);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  :global(main .btn.quiet.on) {
    color: var(--text);
    background: var(--lift);
  }
  footer {
    padding: 5px 14px;
    border-top: 1px solid var(--line);
    color: var(--text-3);
    font-size: 11.5px;
    font-variant-numeric: tabular-nums;
  }
  .pad {
    padding: 8px 14px 0;
  }
</style>
