<script lang="ts">
  import { fade } from "svelte/transition";
  import EntryRow from "../EntryRow.svelte";
  import Icon from "../Icon.svelte";
  import ResultRow from "../ResultRow.svelte";
  import SearchGlyph from "../SearchGlyph.svelte";
  import { SCENES, type Scene } from "./demo";

  interface Props {
    scenes?: Scene[];
    /** Moves on to the next example by itself, until a chip is pressed. */
    autoplay?: boolean;
    /** The chips that jump to an example, and the line under the bar. */
    chips?: boolean;
    /** Room for the card at its tallest (a bar and two rows), so the screen does not jump between examples. */
    height?: number;
  }
  let { scenes = SCENES, autoplay = true, chips = true, height = 206 }: Props = $props();

  // Reduced motion: no typing and no moving on; the example is simply there and the chips choose another.
  const still = typeof matchMedia !== "undefined" && matchMedia("(prefers-reduced-motion: reduce)").matches;

  let index = $state(0);
  let typed = $state(0);
  let open = $state(false);
  // svelte-ignore state_referenced_locally
  let playing = $state(autoplay && !still);
  const scene = $derived(scenes[index]);
  const noop = () => {};

  // Types the example a key at a time, lets the answer pour in, holds it long enough to read, then moves on.
  $effect(() => {
    const { query } = scene;
    const timers: ReturnType<typeof setTimeout>[] = [];
    if (still) {
      typed = query.length;
      open = true;
      return;
    }
    open = false;
    typed = 0;
    let n = 0;
    const type = () => {
      typed = ++n;
      if (n < query.length) {
        timers.push(setTimeout(type, 42 + Math.random() * 60));
        return;
      }
      timers.push(setTimeout(() => (open = true), 240));
      if (playing) timers.push(setTimeout(() => (index = (index + 1) % scenes.length), 4600));
    };
    timers.push(setTimeout(type, 420));
    return () => timers.forEach(clearTimeout);
  });

  function jump(i: number) {
    playing = false;
    index = i;
  }
</script>

<div class="demo" style:height="{height}px">
  <div class="halo" aria-hidden="true"></div>
  <div class="stage" aria-hidden="true">
    <div class="bar">
      <span class="glyph"><SearchGlyph size={22} /></span>
      <span class="text">
        {#if typed === 0}<span class="placeholder">Search apps, files and the web</span>{/if}{scene.query.slice(0, typed)}<i class="caret"></i>
      </span>
    </div>
    <div class="results" class:open>
      <div class="inner">
        {#if open}
          <div class="list" role="presentation">
            {#each scene.files ?? [] as row, i (row.Name)}
              <div class="pour" style:--i={i}>
                <ResultRow {row} query={scene.query} id="demo-file-{i}" selected={i === 0} onhover={noop} onactivate={noop} />
              </div>
            {/each}
            {#each scene.items ?? [] as entry, i (entry.Title)}
              <div class="pour" style:--i={i + (scene.files?.length ?? 0)}>
                <EntryRow {entry} id="demo-item-{i}" selected={!scene.files && i === 0} onhover={noop} onactivate={noop} />
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

{#if chips}
  <div class="chips" role="group" aria-label="Examples">
    {#each scenes as s, i (s.id)}
      <button type="button" class="chip" class:on={i === index} aria-pressed={i === index} onclick={() => jump(i)}>
        <Icon name={s.icon} size={15} />{s.label}
      </button>
    {/each}
  </div>
  {#key index}
    <p class="caption" aria-live="polite" in:fade={{ duration: 220 }}>{scene.caption}</p>
  {/key}
{/if}

<style>
  .demo {
    position: relative;
  }
  /* A slow breath of the accent behind the bar, as the Glass bar's blobs drift: opacity only, which the compositor runs alone. */
  .halo {
    position: absolute;
    inset: -6px -28px 28px;
    background: radial-gradient(ellipse at 50% 22%, color-mix(in srgb, var(--chrome) 26%, transparent), transparent 66%);
    opacity: 0.5;
    will-change: opacity;
    animation: breathe 6s ease-in-out infinite;
    pointer-events: none;
  }
  @keyframes breathe {
    50% {
      opacity: 0.95;
    }
  }
  .stage {
    position: relative;
    overflow: hidden;
    border-radius: 30px;
    background: color-mix(in srgb, var(--ink) 4%, var(--plate));
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, var(--ink) 11%, transparent),
      inset 0 1px 0 color-mix(in srgb, var(--ink) 16%, transparent),
      var(--shadow);
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 14px;
    height: 60px;
    padding: 3px 24px 0;
  }
  .glyph {
    display: grid;
    color: var(--chrome);
  }
  .text {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    white-space: pre;
    color: var(--text);
    font: 400 19px/1 var(--display);
    letter-spacing: -0.01em;
  }
  .placeholder {
    color: var(--text-3);
  }
  .caret {
    display: inline-block;
    width: 2px;
    height: 22px;
    margin: 0 0 -4px 1px;
    border-radius: 1px;
    background: var(--chrome);
    animation: blink 1.05s steps(1) infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }
  .results {
    display: grid;
    grid-template-rows: 0fr;
    transition: grid-template-rows 320ms var(--ease);
  }
  .results.open {
    grid-template-rows: 1fr;
  }
  .inner {
    min-height: 0;
    overflow: hidden;
  }
  .list {
    padding: 2px 0 8px;
    border-top: 1px solid var(--line);
  }
  /* The answer pours in, sliding down out of a small blur, a row after another. */
  .pour {
    animation: pour 420ms var(--ease) both;
    animation-delay: calc(var(--i) * 50ms);
  }
  @keyframes pour {
    from {
      opacity: 0;
      transform: translateY(-6px);
      filter: blur(6px);
    }
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 5px;
    /* a little wider than the column, so the seven chips share one line */
    margin: 0 -24px 12px;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 6px 10px 6px 8px;
    border: 0;
    border-radius: 999px;
    background: var(--tile);
    color: var(--text-2);
    font: 560 12px/1 var(--sans);
    cursor: pointer;
    transition:
      background 160ms var(--ease),
      color 160ms var(--ease),
      box-shadow 160ms var(--ease);
  }
  .chip:hover {
    background: var(--lift);
    color: var(--text);
  }
  .chip.on {
    background: color-mix(in srgb, var(--chrome) 16%, transparent);
    color: var(--text);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chrome) 50%, transparent);
  }
  .caption {
    min-height: 2.8em;
    margin: 0;
    color: var(--text-2);
    font-size: 13px;
    line-height: 1.45;
    text-align: center;
    text-wrap: balance;
  }
  @media (prefers-reduced-motion: reduce) {
    .halo,
    .caret,
    .pour {
      animation: none;
    }
    .results {
      transition: none;
    }
  }
</style>
