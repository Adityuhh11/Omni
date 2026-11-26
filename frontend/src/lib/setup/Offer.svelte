<script lang="ts">
  import type { Snippet } from "svelte";
  import Icon from "../Icon.svelte";

  interface Props {
    icon: "clip" | "chat" | "search";
    title: string;
    blurb: string;
    /** Whether it is switched on: its icon is lit in the accent. */
    on: boolean;
    /** The switch or button at the right. */
    control: Snippet;
    /** What shows under the heading: progress, costs, a problem. */
    detail?: Snippet;
    /** Whether there is anything under the heading right now. */
    showDetail?: boolean;
  }
  let { icon, title, blurb, on, control, detail, showDetail = true }: Props = $props();
</script>

<!-- One row of a list (the list's container draws the box), so three offers are one block, not three. -->
<section class="offer" class:on>
  <div class="top">
    <span class="tile"><Icon name={icon} size={18} /></span>
    <div class="words">
      <h4>{title}</h4>
      <p>{blurb}</p>
    </div>
    <div class="control">{@render control()}</div>
  </div>
  {#if detail && showDetail}<div class="detail">{@render detail()}</div>{/if}
</section>

<style>
  .offer {
    padding: 12px 14px;
  }
  .top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .tile {
    display: grid;
    place-items: center;
    flex: none;
    width: 34px;
    height: 34px;
    border-radius: 10px;
    background: var(--tile);
    color: var(--text-3);
    transition: color 300ms var(--ease);
  }
  .on .tile {
    color: var(--chrome);
  }
  .words {
    flex: 1;
    min-width: 0;
  }
  h4 {
    margin: 0 0 2px;
    font: 600 13.5px/1.25 var(--sans);
  }
  p {
    margin: 0;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.4;
  }
  .control {
    display: flex;
    flex: none;
    align-items: center;
    gap: 8px;
  }
  .detail {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin: 12px 0 0 46px;
  }
</style>
