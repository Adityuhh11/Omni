<script lang="ts">
  import type { Snippet } from "svelte";
  import Icon from "../Icon.svelte";

  interface Props {
    icon: "clip" | "chat" | "search";
    title: string;
    blurb: string;
    /** What is typed in the bar to use it. */
    example: string;
    /** Whether it is switched on: the card is lit in the accent. */
    on: boolean;
    /** The switch or button at the right. */
    control: Snippet;
    /** What shows under the heading: progress, costs, a problem. */
    detail?: Snippet;
    /** Whether there is anything under the heading right now; with nothing, the card has no divider either. */
    showDetail?: boolean;
  }
  let { icon, title, blurb, example, on, control, detail, showDetail = true }: Props = $props();
</script>

<section class="offer" class:on>
  <div class="top">
    <span class="tile"><Icon name={icon} /></span>
    <div class="words">
      <h4>{title}</h4>
      <p>{blurb}</p>
      <kbd>{example}</kbd>
    </div>
    <div class="control">{@render control()}</div>
  </div>
  {#if detail && showDetail}<div class="detail">{@render detail()}</div>{/if}
</section>

<style>
  .offer {
    padding: 16px;
    border-radius: 16px;
    background: var(--tile);
    box-shadow: inset 0 0 0 1px var(--line);
    transition:
      background 300ms var(--ease),
      box-shadow 300ms var(--ease);
  }
  .offer.on {
    background: color-mix(in srgb, var(--chrome) 7%, var(--tile));
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 42%, transparent);
  }
  .top {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .tile {
    display: grid;
    place-items: center;
    flex: none;
    width: 44px;
    height: 44px;
    border-radius: 13px;
    background: var(--tile);
    color: var(--text-2);
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
    margin: 0 0 3px;
    font: 620 14px/1.25 var(--sans);
  }
  p {
    margin: 0 0 7px;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.45;
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
    gap: 12px;
    margin-top: 14px;
    padding-top: 14px;
    border-top: 1px solid var(--line);
  }
</style>
