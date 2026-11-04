<script lang="ts">
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { View } from "../../../bindings/omni/internal/settingsui";

  let { view }: { view: View } = $props();

  type Row = { trigger: string; title: string; url: string };
  const complete = (r: Row) => r.trigger.trim() !== "" && r.title.trim() !== "" && r.url.trim() !== "";
  const body = (list: Row[]) =>
    list.filter(complete).map((r) => ({ Trigger: r.trigger.trim(), Title: r.title.trim(), URL: r.url.trim() }));

  // The table starts from the quicklinks as they were when the page opened and then owns its rows.
  // svelte-ignore state_referenced_locally
  let rows = $state<Row[]>((view.Quicklinks ?? []).map((l) => ({ trigger: l.Trigger, title: l.Title, url: l.URL })));
  let lastSaved = JSON.stringify(body(rows));
  let error = $state("");
  let saved = $state(false);
  let savedTimer: ReturnType<typeof setTimeout> | undefined;
  const incomplete = $derived(rows.some((r) => !complete(r) && (r.trigger || r.title || r.url)));
  const QUERY = "{q}";

  // A row is saved once all three of its fields are filled in; until then it stays on this page only.
  async function commit() {
    const links = body(rows);
    const text = JSON.stringify(links);
    if (text === lastSaved) return;
    error = await Settings.SaveQuicklinks(links);
    if (error !== "") return;
    lastSaved = text;
    saved = true;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => (saved = false), 1800);
  }

  function remove(index: number) {
    rows.splice(index, 1);
    commit();
  }
</script>

<div class="head-row">
  <h2>Quicklinks</h2>
  <p class="ok" aria-live="polite">{saved ? "Saved" : ""}</p>
</div>

<h3 class="group-title">Type a trigger and words: <span class="mono">gh rust lifetimes</span></h3>
<div class="group" role="group" aria-label="Quicklinks">
  {#if rows.length > 0}
    <div class="item cols head" aria-hidden="true"><span>Trigger</span><span>Name</span><span>Address, with {QUERY} for the words</span><span></span></div>
    {#each rows as row, i (i)}
      <div class="item cols">
        <input type="text" class="mono" aria-label="Trigger" placeholder="gh" bind:value={row.trigger} onchange={commit} spellcheck="false" />
        <input type="text" aria-label="Name" placeholder="GitHub" bind:value={row.title} onchange={commit} />
        <input type="text" class="mono" aria-label="Address" placeholder={"https://github.com/search?q=" + QUERY} bind:value={row.url} onchange={commit} spellcheck="false" />
        <button type="button" class="btn quiet" aria-label="Remove this quicklink" onclick={() => remove(i)}>✕</button>
      </div>
    {/each}
  {:else}
    <p class="item sub">No quicklinks yet.</p>
  {/if}
  <div class="item">
    <button type="button" class="btn" onclick={() => rows.push({ trigger: "", title: "", url: "" })}>Add quicklink</button>
    {#if incomplete}<span class="sub">Saved once all three are filled in.</span>{/if}
  </div>
</div>
{#if error}<p class="error">{error}</p>{/if}

<style>
  .cols {
    display: grid;
    grid-template-columns: 84px 120px minmax(0, 1fr) 34px;
    gap: 8px;
  }
  .item.cols > input {
    width: auto;
    min-width: 0;
  }
  .head {
    min-height: 0;
    padding-bottom: 0;
    color: var(--text-3);
    font: 500 11.5px/1 var(--sans);
  }
  .head + .item {
    border-top: 0;
  }
</style>
