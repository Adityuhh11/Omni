<script lang="ts">
  import { onMount } from "svelte";

  /** running: the bar is open. Closed, the lights finish their drift and wait. */
  let { running }: { running: boolean } = $props();

  const still = matchMedia("(prefers-reduced-motion: reduce)");
  let lights: HTMLElement[] = $state([]);

  /** Sends a light, over seconds, to a random place in the card, size and strength. */
  function drift(light: HTMLElement, seconds: number) {
    light.style.transitionDuration = `${seconds}s`;
    light.style.transform = `translate(${Math.random() * 100}cqw, ${Math.random() * 100}cqh) scale(${0.7 + Math.random() * 0.7})`;
    light.style.opacity = String(0.55 + Math.random() * 0.45);
  }

  onMount(() => lights.forEach((light) => drift(light, 0))); // somewhere in the card from the start, not in its corner

  $effect(() => {
    if (!running || still.matches) return;
    const timers = lights.map((light) => {
      let timer = 0;
      const next = () => {
        const seconds = 6 + Math.random() * 6;
        drift(light, seconds);
        timer = window.setTimeout(next, seconds * 1000);
      };
      next();
      return () => clearTimeout(timer);
    });
    return () => timers.forEach((stop) => stop());
  });
</script>

<!--
  Two faint lights in the accent's colour wandering inside the bar, each to a random place at its own pace: switched on in
  Settings, in the Glass theme while the system blurs (data-glass unset). One style change every few seconds; the compositor
  draws the drift. The parent isolates its stacking, so the lights lie over its plate and under its content.
-->
<div class="glow" aria-hidden="true">
  <span class="light" bind:this={lights[0]}></span>
  <span class="light" bind:this={lights[1]}></span>
</div>

<style>
  .glow {
    position: absolute;
    inset: 0;
    z-index: -1;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    container-type: size;
  }
  :global(html:not([data-theme="glass"])) .glow,
  :global(html:not([data-lights="on"])) .glow,
  :global(html[data-glass]) .glow {
    display: none;
  }
  .light {
    /* Wide, soft ovals: a band of light rather than a dot. */
    --w: max(260px, 34cqw);
    --h: 190px;
    position: absolute;
    left: calc(var(--w) / -2);
    top: calc(var(--h) / -2);
    width: var(--w);
    height: var(--h);
    border-radius: 50%;
    background: radial-gradient(closest-side, color-mix(in srgb, var(--chrome) 24%, transparent), transparent);
    transition-property: transform, opacity;
    transition-timing-function: ease-in-out;
    will-change: transform;
  }
</style>
