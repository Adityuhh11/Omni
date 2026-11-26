<script lang="ts">
  import { onMount } from "svelte";
  import { hexToHsv, hsvToHex, normalizeHex, type Hsv } from "./color";

  /** `value` is the colour it opens on. `onpreview` is told every change while the pointer is down; `oncommit` once a change is final. */
  let { value, onpreview, oncommit, onclose }: { value: string; onpreview: (hex: string) => void; oncommit: (hex: string) => void; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  let hsv = $state<Hsv>(hexToHsv(value));
  // svelte-ignore state_referenced_locally
  let text = $state(value);
  let area = $state<HTMLDivElement | undefined>();
  let root = $state<HTMLDivElement | undefined>();
  let dragging = false;
  let above = $state(false); // no room under the swatches: open over them instead

  const hex = $derived(hsvToHex(hsv));
  const clamp = (n: number) => Math.min(1, Math.max(0, n));

  onMount(() => {
    if (root && root.getBoundingClientRect().bottom > innerHeight - 12) above = true;
  });

  function changed() {
    text = hex;
    onpreview(hex);
  }

  function pick(event: PointerEvent) {
    if (!area) return;
    const box = area.getBoundingClientRect();
    hsv.s = clamp((event.clientX - box.left) / box.width);
    hsv.v = 1 - clamp((event.clientY - box.top) / box.height);
    changed();
  }

  function down(event: PointerEvent) {
    dragging = true;
    area?.setPointerCapture(event.pointerId);
    pick(event);
  }

  function up() {
    if (!dragging) return;
    dragging = false;
    oncommit(hex);
  }

  function arrows(event: KeyboardEvent) {
    const step = event.shiftKey ? 0.1 : 0.01;
    const move = { ArrowLeft: [-step, 0], ArrowRight: [step, 0], ArrowUp: [0, step], ArrowDown: [0, -step] }[event.key];
    if (!move) return;
    event.preventDefault();
    hsv.s = clamp(hsv.s + move[0]);
    hsv.v = clamp(hsv.v + move[1]);
    changed();
    oncommit(hex);
  }

  function typed(event: Event & { currentTarget: HTMLInputElement }) {
    text = event.currentTarget.value;
    const found = normalizeHex(text);
    if (!found) return;
    hsv = hexToHsv(found);
    onpreview(found);
  }

  function typedDone() {
    const found = normalizeHex(text);
    if (found) oncommit(found);
    else text = hex; // what was typed is not a colour: back to the one shown
  }

  function outside(event: PointerEvent) {
    if (root && !root.contains(event.target as Node)) onclose();
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    event.stopPropagation();
    onclose();
  }
</script>

<svelte:window onpointerdown={outside} onkeydown={onKeydown} />

<div class="picker" class:above bind:this={root} role="dialog" aria-label="Pick a colour">
  <div
    class="field"
    bind:this={area}
    style:--hue={hsv.h}
    role="slider"
    tabindex="0"
    aria-label="Saturation and brightness"
    aria-valuetext="Saturation {Math.round(hsv.s * 100)} percent, brightness {Math.round(hsv.v * 100)} percent"
    aria-valuenow={Math.round(hsv.s * 100)}
    onpointerdown={down}
    onpointermove={(event) => dragging && pick(event)}
    onpointerup={up}
    onpointercancel={up}
    onkeydown={arrows}
  >
    <span class="thumb" style:left="{hsv.s * 100}%" style:top="{(1 - hsv.v) * 100}%" style:background={hex}></span>
  </div>
  <input class="hue" type="range" min="0" max="360" step="1" aria-label="Hue" bind:value={hsv.h} oninput={changed} onchange={() => oncommit(hex)} />
  <div class="row">
    <span class="now" style:background={hex} aria-hidden="true"></span>
    <input class="hex" type="text" aria-label="Colour as #rrggbb" spellcheck="false" autocomplete="off" maxlength="7" value={text} oninput={typed} onchange={typedDone} onkeydown={(event) => event.key === "Enter" && typedDone()} />
    <button type="button" class="done" onclick={onclose}>Done</button>
  </div>
</div>

<style>
  .picker {
    position: absolute;
    top: calc(100% + 8px);
    left: 0;
    z-index: 20;
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 232px;
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: 14px;
    background: var(--glass-raised);
    -webkit-backdrop-filter: var(--glass-blur);
    backdrop-filter: var(--glass-blur);
    box-shadow: 0 16px 36px -10px rgba(0, 0, 0, 0.55), 0 2px 6px rgba(0, 0, 0, 0.3);
    animation: pop 120ms var(--ease) backwards;
  }
  .picker.above {
    top: auto;
    bottom: calc(100% + 8px);
    animation-name: pop-up;
  }
  @keyframes pop-up {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
  }
  .field {
    position: relative;
    height: 132px;
    border-radius: 9px;
    background:
      linear-gradient(to top, #000, transparent),
      linear-gradient(to right, #fff, hsl(calc(var(--hue) * 1deg) 100% 50%));
    cursor: crosshair;
    touch-action: none;
  }
  .field:focus-visible {
    outline: 2px solid var(--chrome);
    outline-offset: 2px;
  }
  .thumb {
    position: absolute;
    width: 16px;
    height: 16px;
    border: 2px solid #fff;
    border-radius: 50%;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35), 0 2px 5px rgba(0, 0, 0, 0.4);
    transform: translate(-50%, -50%);
    pointer-events: none;
  }
  .hue {
    width: 100%;
    height: 12px;
    margin: 0;
    border-radius: 6px;
    background: linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00);
    appearance: none;
    cursor: pointer;
  }
  .hue::-webkit-slider-runnable-track {
    height: 12px;
    background: transparent;
  }
  .hue::-webkit-slider-thumb {
    margin-top: -2px;
    width: 16px;
    height: 16px;
    border: 2px solid #fff;
    border-radius: 50%;
    background: transparent;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35), 0 2px 5px rgba(0, 0, 0, 0.4);
    appearance: none;
  }
  .hue:focus-visible {
    outline: 2px solid var(--chrome);
    outline-offset: 3px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .now {
    flex: none;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink) 22%, transparent);
  }
  /* beats the form styles of main input[type="text"], which this field is inside */
  .row .hex[type="text"] {
    flex: 1;
    min-width: 0;
    height: 30px;
    padding: 0 9px;
    border-radius: 8px;
    font: 500 12.5px/1 var(--mono);
    text-transform: lowercase;
  }
  .done {
    flex: none;
    height: 30px;
    padding: 0 12px;
    border: 0;
    border-radius: 8px;
    background: var(--chrome);
    color: var(--on-chrome);
    font: 600 12.5px/1 var(--sans);
    cursor: pointer;
  }
  .done:hover {
    background: var(--chrome-hot);
  }
  .done:focus-visible {
    outline: 2px solid var(--chrome);
    outline-offset: 2px;
  }
  @media (prefers-reduced-motion: reduce) {
    .picker {
      animation: none;
    }
  }
</style>
