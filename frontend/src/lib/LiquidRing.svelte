<script lang="ts">
  import { onMount } from "svelte";
  import { ShaderMount, getShaderColorFromString } from "@paper-design/shaders";

  import type { Metal } from "./palettes";
  import { ringShader } from "./ringShader";

  interface Props {
    /** The flow speed to settle at; the ring eases there instead of jumping. */
    speed: number;
    /** False while the panel is closed: the ring stops, and with it every animation frame it costs. */
    running: boolean;
    /** What the metal is made of; it can change while the ring runs. */
    metal: Metal;
    /** The corner radius in px of the outline the light follows: the search card's by default. */
    radius?: number;
  }
  let { speed, running, metal, radius = 30 }: Props = $props();

  // A change of speed takes about this long to settle. Typing changes the speed many times a second,
  // and a ring that lurches with every keystroke looks broken.
  const EASE_MS = 350;

  // The shader fills the whole card and is masked to its outer rim (--ring wide), so what shows is a ring of flowing metal and the
  // glass inside stays clear of it.

  const reduceMotion = matchMedia("(prefers-reduced-motion: reduce)").matches;

  let host: HTMLDivElement;
  let mount: ShaderMount | undefined;

  onMount(() => {
    try {
      mount = new ShaderMount(
        host,
        ringShader,
        {
          u_colorBack: getShaderColorFromString(metal.back),
          u_colorTint: getShaderColorFromString(metal.tint),
          // Read once: a ring does not change its shape (a composer that grows keeps its corners).
          // svelte-ignore state_referenced_locally
          u_radius: radius,
        },
        undefined,
        0, // stopped until the ring is told it is running
        0,
        // The ring is a few pixels wide: one pixel per CSS pixel is plenty, and the GPU is shared with
        // the embedding model.
        1,
        700_000,
      );
    } catch (error) {
      // No WebGL2: the card's static chrome gradient stands in for the ring.
      console.warn("liquid ring unavailable:", error);
    }
    return () => mount?.dispose();
  });

  // Eases the shader's speed toward `speed` while running, one step per frame, and stops asking for
  // frames once it has arrived. Closing the panel stops the shader at once (nobody sees it stop).
  let current = 0;
  let frame = 0;
  $effect(() => {
    const target = running && !reduceMotion ? speed : 0;
    cancelAnimationFrame(frame);
    if (!mount) return;
    if (target === 0) {
      current = 0;
      mount.setSpeed(0);
      return;
    }
    let last = performance.now();
    const step = (now: number) => {
      current += (target - current) * (1 - Math.exp(-(now - last) / EASE_MS));
      last = now;
      if (Math.abs(target - current) < 0.01) current = target;
      mount?.setSpeed(current);
      if (current !== target) frame = requestAnimationFrame(step);
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  });
  // The colours are read before `mount` is looked at: `mount?.setUniforms(...)` would skip reading them
  // while there is no mount yet, and an effect that read nothing never runs again.
  $effect(() => {
    const back = getShaderColorFromString(metal.back);
    const tint = getShaderColorFromString(metal.tint);
    mount?.setUniforms({ u_colorBack: back, u_colorTint: tint });
  });
</script>

<div class="ring" bind:this={host} aria-hidden="true"></div>

<style>
  .ring {
    position: absolute;
    inset: 0;
    box-sizing: border-box;
    padding: var(--ring);
    border-radius: inherit;
    overflow: hidden;
    /* The ring's first frame, and its stand-in when the shader cannot run: static chrome. */
    background: var(--ring-fallback);
    -webkit-mask-image: linear-gradient(#000 0 0), linear-gradient(#000 0 0);
    -webkit-mask-clip: content-box, border-box;
    -webkit-mask-composite: xor;
    mask-image: linear-gradient(#000 0 0), linear-gradient(#000 0 0);
    mask-clip: content-box, border-box;
    mask-composite: exclude;
  }
  .ring :global(canvas) {
    position: absolute; /* over the padding too: the mask, not the box, keeps it to the rim */
    inset: 0;
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
