<script lang="ts">
  import { nextDrift, type Drift } from "./drift";

  /** running: the bar is open. Closed, the lights finish their stretch and wait. */
  let { running }: { running: boolean } = $props();

  const still = matchMedia("(prefers-reduced-motion: reduce)");
  let lights: HTMLElement[] = $state([]);

  /** Puts a light at a point, getting there in seconds at a constant pace, a little larger or smaller and brighter or fainter. */
  function place(light: HTMLElement, at: Drift, seconds: number) {
    light.style.transitionDuration = `${seconds}s`;
    light.style.transform = `translate(${at.x}px, ${at.y}px) scale(${0.8 + Math.random() * 0.4})`;
    light.style.opacity = String(0.6 + Math.random() * 0.4);
  }

  $effect(() => {
    const box = lights[0]?.parentElement;
    if (!box) return;
    const paths = lights.map((light) => {
      const at: Drift = { x: Math.random() * box.clientWidth, y: Math.random() * box.clientHeight, heading: Math.random() * 2 * Math.PI };
      place(light, at, 0);
      return at;
    });
    if (!running || still.matches) return;
    const timers = lights.map((light, i) => {
      let timer = 0;
      const next = () => {
        const step = nextDrift(paths[i], box.clientWidth, box.clientHeight);
        paths[i] = step;
        place(light, step, step.seconds);
        timer = window.setTimeout(next, step.seconds * 1000);
      };
      next();
      return () => clearTimeout(timer);
    });
    return () => timers.forEach((stop) => stop());
  });
</script>

<!--
  Two faint lights in the accent's colour drifting inside the bar like a slow current, at one constant pace (lib/drift.ts): switched on in
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
    transition-timing-function: linear;
    will-change: transform;
  }
</style>
