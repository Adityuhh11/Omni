<script lang="ts">
  import { onMount } from "svelte";
  import { EditorState, EditorView, liveEditor } from "./live";

  /** `text` is what the editor starts with: a new note gets a new editor (and its own undo history). `onchange` is told the whole text. */
  let { text, placeholder, onchange, openLink }: { text: string; placeholder: string; onchange: (text: string) => void; openLink: (href: string) => void } = $props();

  let host: HTMLDivElement;
  let view: EditorView | undefined;

  /** Puts the caret in the note (at its end, where writing goes on). */
  export function focus() {
    view?.focus();
  }

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        // svelte-ignore state_referenced_locally
        doc: text,
        selection: { anchor: text.length },
        extensions: [
          liveEditor({ placeholder, openLink, label: "Note" }),
          EditorView.updateListener.of((update) => update.docChanged && onchange(update.state.doc.toString())),
        ],
      }),
    });
    return () => view?.destroy();
  });
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
