<script lang="ts">
  import { AGE_RAMP } from "./folders";

  export interface Entry {
    name: string; // what the row says
    title: string; // the full name, on hover
    color: string;
    count: number;
  }

  let {
    label,
    entries = [],
    hidden,
    cuts = null,
    total,
    ontoggle,
  }: {
    label: string;
    entries?: Entry[];
    hidden: Set<number>;
    cuts?: [number, number, number] | null; // by age: the oldest, middle and newest dates (seconds)
    total: number;
    ontoggle: (index: number) => void;
  } = $props();

  const date = (seconds: number) =>
    seconds ? new Date(seconds * 1000).toLocaleDateString(undefined, { year: "numeric", month: "short" }) : "?";
</script>

<div class="legend" aria-label={label} role="group">
  {#if cuts}
    <div class="ramp" style:background="linear-gradient(to right, {AGE_RAMP.join(', ')})"></div>
    <div class="dates">
      <span>{date(cuts[0])}</span><span>{date(cuts[1])}</span><span>{date(cuts[2])}</span>
    </div>
    <p class="note">Older to newer, by when each file last changed</p>
  {:else}
    <ul>
      {#each entries as entry, index}
        <li>
          <button class:off={hidden.has(index)} onclick={() => ontoggle(index)} title={entry.title} aria-pressed={!hidden.has(index)}>
            <span class="dot" style:background={entry.color}></span>
            <span class="name">{entry.name}</span>
            <span class="count">{entry.count.toLocaleString()}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
  <p class="total">{total.toLocaleString()} pieces</p>
</div>

<style>
  .legend {
    position: absolute;
    left: 16px;
    bottom: 16px;
    max-height: 50vh;
    overflow-y: auto;
    padding: 8px;
    border: 1px solid var(--edge);
    border-radius: 14px;
    background: var(--panel);
    backdrop-filter: blur(12px);
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  button {
    display: flex;
    align-items: center;
    gap: 9px;
    width: 100%;
    padding: 5px 8px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--ink-2);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  button:hover {
    background: rgba(255, 255, 255, 0.06);
  }
  button.off {
    color: var(--ink-3);
  }
  button.off .dot {
    opacity: 0.25;
  }
  .dot {
    flex: none;
    width: 9px;
    height: 9px;
    border-radius: 50%;
  }
  .name {
    flex: 1;
    max-width: 220px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .count,
  .total,
  .dates,
  .note {
    color: var(--ink-3);
    font: 400 11px var(--mono);
  }
  .total {
    margin: 0;
    padding: 6px 8px 2px;
  }
  .ramp {
    width: 240px;
    height: 10px;
    margin: 6px 8px 4px;
    border-radius: 5px;
  }
  .dates {
    display: flex;
    justify-content: space-between;
    width: 240px;
    margin: 0 8px;
  }
  .note {
    margin: 6px 8px 0;
    font-family: var(--sans);
  }
</style>
