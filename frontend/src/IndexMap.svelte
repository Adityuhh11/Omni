<script lang="ts">
  import { onMount } from "svelte";
  import { Service as Atlas } from "../bindings/omni/internal/atlas";
  import type { View } from "../bindings/omni/internal/atlas";
  import type { MapPoint } from "../bindings/omni/internal/sidecar";
  import Loader from "./lib/Loader.svelte";
  import { createRenderer, type Renderer } from "./lib/map/gl";
  import { colorsFor, folderName, SLOTS } from "./lib/map/folders";
  import {
    approach,
    clamp,
    decodeFloats,
    decodeUint16,
    FOV,
    MAX_DISTANCE,
    MAX_PITCH,
    MIN_DISTANCE,
    pick,
    toScreen,
    viewProjection,
    type Camera,
  } from "./lib/map/view";

  const HOME: Camera = { yaw: 0.6, pitch: 0.35, distance: 2.6, target: [0, 0, 0], flat: 0 };
  const IDLE_MS = 4000; // after this long untouched, the 3D map turns slowly by itself
  const SPIN = 0.06; // radians per second
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;

  let view = $state<View | null>(null);
  let problem = $state("");
  let canvas: HTMLCanvasElement;
  let renderer: Renderer | null = null;
  let noWebGL = $state(false);

  // The map itself.
  let coords: Float32Array | null = null;
  let source: Uint16Array | null = null;
  let roots = $state<string[]>([]);
  let counts = $state<number[]>([]);
  let hidden = $state(new Set<number>());

  // The camera eases toward the goal every frame; input moves the goal.
  const now: Camera = structuredClone(HOME);
  const goal: Camera = structuredClone(HOME);
  let flatView = $state(false);
  let lastInput = performance.now();

  // Hover and query.
  let hover = $state<{ index: number; x: number; y: number; point: MapPoint | null } | null>(null);
  let query = $state("");
  let located = $state<{ at: [number, number, number]; points: number[] } | null>(null);
  let marker = $state<{ x: number; y: number } | null>(null);
  let dim = 1;
  let searching = $state(false);

  // --- loading -------------------------------------------------------------------------------------------------------

  async function refresh() {
    try {
      view = await Atlas.State();
      problem = "";
    } catch (error) {
      problem = String(error);
      return;
    }
    if (view.State === "ready" && !coords) await load();
  }

  async function load() {
    try {
      const points = await Atlas.Points();
      coords = decodeFloats(points.coords ?? "");
      source = decodeUint16(points.source ?? "");
      roots = points.roots ?? [];
      const tally = roots.map(() => 0);
      for (const s of source) tally[s]++;
      counts = tally;
      renderer?.setPoints(coords, colorsFor(source, roots, hidden));
      kick();
    } catch (error) {
      problem = "The map could not be read: " + String(error);
    }
  }

  async function build() {
    coords = null;
    located = null;
    await Atlas.Build();
    await refresh();
  }

  // Polled only while something is under way.
  $effect(() => {
    if (!view || !["installing", "building"].includes(view.State)) return;
    const timer = setInterval(refresh, 1000);
    return () => clearInterval(timer);
  });

  // --- drawing ---------------------------------------------------------------------------------------------------------

  let frame = 0;
  let last = 0;

  function kick() {
    if (!frame) {
      last = performance.now();
      frame = requestAnimationFrame(tick);
    }
  }

  function tick(time: number) {
    frame = 0;
    const dt = Math.min(0.1, (time - last) / 1000);
    last = time;
    const idle = !reduced && !flatView && time - lastInput > IDLE_MS && !hover;
    if (idle) goal.yaw += SPIN * dt;
    const rate = reduced ? 1000 : 9;
    let moving = idle;
    for (const key of ["yaw", "pitch", "distance", "flat"] as const) {
      now[key] = approach(now[key], goal[key], dt, rate);
      moving ||= now[key] !== goal[key];
    }
    for (let i = 0; i < 3; i++) {
      now.target[i] = approach(now.target[i], goal.target[i], dt, rate);
      moving ||= now.target[i] !== goal.target[i];
    }
    const dimGoal = located ? 0.22 : 1;
    dim = approach(dim, dimGoal, dt, rate);
    moving ||= dim !== dimGoal;
    draw();
    if (moving && !document.hidden) kick();
  }

  function matrix() {
    return viewProjection(now, canvas.width / Math.max(1, canvas.height));
  }

  function draw() {
    if (!renderer || !coords) return;
    const m = matrix();
    const dpr = devicePixelRatio || 1;
    renderer.draw(m, now.flat, dim, 8.5 * dpr * Math.sqrt(30000 / Math.max(2000, coords.length / 3)));
    if (located) {
      const at = toScreen(m, located.at[0], located.at[1], located.at[2] * (1 - now.flat), canvas.width, canvas.height);
      marker = at && { x: at[0] / dpr, y: at[1] / dpr };
    } else marker = null;
  }

  function resize() {
    const dpr = devicePixelRatio || 1;
    canvas.width = Math.round(canvas.clientWidth * dpr);
    canvas.height = Math.round(canvas.clientHeight * dpr);
    draw();
  }

  // --- input ---------------------------------------------------------------------------------------------------------

  let drag: { x: number; y: number; pan: boolean; moved: boolean } | null = null;

  function touched() {
    lastInput = performance.now();
    kick();
  }

  function onpointerdown(e: PointerEvent) {
    canvas.setPointerCapture(e.pointerId);
    drag = { x: e.clientX, y: e.clientY, pan: e.button === 2 || e.shiftKey || flatView, moved: false };
    touched();
  }

  function onpointermove(e: PointerEvent) {
    if (drag) {
      const dx = e.clientX - drag.x;
      const dy = e.clientY - drag.y;
      drag.x = e.clientX;
      drag.y = e.clientY;
      drag.moved ||= Math.abs(dx) + Math.abs(dy) > 2;
      if (drag.pan) pan(dx, dy);
      else {
        goal.yaw -= dx * 0.006;
        goal.pitch = clamp(goal.pitch + dy * 0.006, -MAX_PITCH, MAX_PITCH);
      }
      hover = null;
      touched();
      return;
    }
    pickAt(e.clientX, e.clientY);
  }

  function onpointerup() {
    const clicked = drag && !drag.moved;
    drag = null;
    if (clicked && hover) Atlas.Open(hover.index).catch((error) => (problem = String(error)));
  }

  function pan(dx: number, dy: number) {
    const perPixel = (2 * goal.distance * Math.tan(FOV / 2)) / canvas.clientHeight;
    const right = [Math.cos(goal.yaw), 0, -Math.sin(goal.yaw)];
    const up = [-Math.sin(goal.pitch) * Math.sin(goal.yaw), Math.cos(goal.pitch), -Math.sin(goal.pitch) * Math.cos(goal.yaw)];
    for (let i = 0; i < 3; i++) goal.target[i] += (-dx * right[i] + dy * up[i]) * perPixel;
  }

  function onwheel(e: WheelEvent) {
    e.preventDefault();
    goal.distance = clamp(goal.distance * Math.exp(e.deltaY * 0.0012), MIN_DISTANCE, MAX_DISTANCE);
    touched();
  }

  // Double-click: fly to the point under the pointer.
  function ondblclick() {
    if (!hover || !coords) return;
    const i = hover.index;
    goal.target = [coords[3 * i], coords[3 * i + 1], coords[3 * i + 2] * (1 - goal.flat)];
    goal.distance = Math.max(MIN_DISTANCE, goal.distance * 0.4);
    touched();
  }

  let hoverToken = 0;
  function pickAt(clientX: number, clientY: number) {
    if (!coords) return;
    const rect = canvas.getBoundingClientRect();
    const dpr = devicePixelRatio || 1;
    const index = pick(matrix(), coords, now.flat, (i) => !hidden.has(source![i]), (clientX - rect.left) * dpr,
      (clientY - rect.top) * dpr, canvas.width, canvas.height, 10 * dpr);
    if (index < 0) {
      hover = null;
      return;
    }
    if (hover?.index === index) {
      hover = { ...hover, x: clientX, y: clientY };
      return;
    }
    hover = { index, x: clientX, y: clientY, point: null };
    const token = ++hoverToken;
    Atlas.Point(index)
      .then((point) => {
        if (token === hoverToken && hover?.index === index) hover = { ...hover, point };
      })
      .catch(() => {});
  }

  function setFlat(on: boolean) {
    flatView = on;
    goal.flat = on ? 1 : 0;
    if (on) {
      goal.yaw = 0;
      goal.pitch = 0;
    } else {
      goal.yaw = HOME.yaw;
      goal.pitch = HOME.pitch;
    }
    goal.target[2] = on ? 0 : goal.target[2];
    touched();
  }

  function home() {
    Object.assign(goal, structuredClone({ ...HOME, flat: goal.flat, yaw: flatView ? 0 : HOME.yaw, pitch: flatView ? 0 : HOME.pitch }));
    touched();
  }

  function toggleFolder(index: number) {
    const next = new Set(hidden);
    if (!next.delete(index)) next.add(index);
    hidden = next;
    if (coords && source) renderer?.setColors(colorsFor(source, roots, hidden));
    kick();
  }

  async function locate(e: SubmitEvent) {
    e.preventDefault();
    const text = query.trim();
    if (!text || !coords) return clearQuery();
    searching = true;
    try {
      const found = await Atlas.Locate(text);
      const at = found.at as [number, number, number];
      located = { at, points: found.points ?? [] };
      const lit = new Float32Array(coords.length / 3);
      for (const i of located.points) lit[i] = 1;
      renderer?.setLit(lit);
      goal.target = [at[0], at[1], at[2] * (1 - goal.flat)];
      goal.distance = Math.min(goal.distance, 1.1);
      problem = "";
    } catch (error) {
      problem = String(error).includes("404") ? "Nothing on the map is near that." : String(error);
    } finally {
      searching = false;
      touched();
    }
  }

  function clearQuery() {
    located = null;
    if (coords) renderer?.setLit(new Float32Array(coords.length / 3));
    touched();
  }

  function onkeydown(e: KeyboardEvent) {
    if ((e.target as HTMLElement).tagName === "INPUT") {
      if (e.key === "Escape") {
        query = "";
        clearQuery();
      }
      return;
    }
    if (e.key === "Escape") clearQuery();
    else if (e.key === "f" || e.key === "F") home();
    else if (e.key === "2") setFlat(true);
    else if (e.key === "3") setFlat(false);
  }

  onMount(() => {
    renderer = createRenderer(canvas);
    noWebGL = renderer === null;
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    const wake = () => !document.hidden && kick();
    document.addEventListener("visibilitychange", wake);
    refresh();
    return () => {
      observer.disconnect();
      document.removeEventListener("visibilitychange", wake);
      cancelAnimationFrame(frame);
    };
  });

  const ready = $derived(view?.State === "ready" && counts.length > 0);
  const total = $derived(counts.reduce((a, b) => a + b, 0));
</script>

<svelte:window {onkeydown} />

<main>
  <canvas
    bind:this={canvas}
    class:pointing={hover !== null}
    {onpointerdown}
    {onpointermove}
    {onpointerup}
    {onwheel}
    {ondblclick}
    onpointerleave={() => (hover = null)}
    oncontextmenu={(e) => e.preventDefault()}
    aria-label="A map of the index: every dot is a piece of one of your files, near others that mean something alike"
  ></canvas>

  {#if marker}
    <span class="marker" style:left="{marker.x}px" style:top="{marker.y}px" aria-hidden="true"></span>
  {/if}

  {#if ready}
    <header>
      <form onsubmit={locate}>
        <input bind:value={query} placeholder="Where would this land?" aria-label="Find where a search lands on the map" />
        {#if searching}<Loader text="Looking" />{/if}
      </form>
      <div class="toggle" role="group" aria-label="View">
        <button class:on={!flatView} onclick={() => setFlat(false)} aria-pressed={!flatView}>3D</button>
        <button class:on={flatView} onclick={() => setFlat(true)} aria-pressed={flatView}>2D</button>
      </div>
      {#if view?.Stale}
        <button class="quiet" onclick={build} title="The index has changed since this map was made">Rebuild</button>
      {/if}
    </header>

    <ul class="legend" aria-label="Folders">
      {#each roots as root, index}
        <li>
          <button class:off={hidden.has(index)} onclick={() => toggleFolder(index)} title={root || "A folder no longer searched"}
            aria-pressed={!hidden.has(index)}>
            <span class="dot" style:background={index < SLOTS.length && root ? SLOTS[index] : "var(--other)"}></span>
            <span class="name">{folderName(root)}</span>
            <span class="count">{counts[index]?.toLocaleString()}</span>
          </button>
        </li>
      {/each}
      <li class="total">{total.toLocaleString()} pieces</li>
    </ul>

    <p class="help">Drag to turn · right-drag to move · scroll to zoom · double-click to fly in · F to come back</p>
  {/if}

  {#if hover?.point}
    {@const card = hover.point}
    <aside class="card" style:left="{Math.min(hover.x + 16, innerWidth - 336)}px" style:top="{Math.min(hover.y + 16, innerHeight - 200)}px">
      <p class="where">{folderName(card.root)} · {card.kind.replace("_", " ")}</p>
      <p class="file">{card.uri}</p>
      {#if card.text}<p class="text">{card.text}</p>{/if}
      <p class="hint">Click to open</p>
    </aside>
  {/if}

  {#if !ready}
    <section class="state">
      {#if noWebGL}
        <p>This computer cannot draw the map (WebGL2 is not available).</p>
      {:else if !view}
        <Loader text="Opening" />
      {:else if view.State === "installing"}
        <Loader text="Installing" />
        <p class="small">{view.Message}</p>
      {:else if view.State === "building"}
        <Loader text="Laying out" />
        <p class="small">Placing every piece of your files by what it means. About a minute for 50,000.</p>
      {:else if view.State === "off"}
        <p>{view.Message}</p>
      {:else if view.State === "ready"}
        <Loader text="Drawing" />
      {:else}
        <h1>A map of everything you search</h1>
        <p>Every dot is a piece of one of your files. Pieces that mean something alike lie close together.</p>
        {#if view.State === "failed" && view.Message}<p class="problem">{view.Message}</p>{/if}
        <button class="primary" onclick={build}>
          {view.State === "missing" ? "Install and build the map" : "Build the map"}
        </button>
        {#if view.State === "missing"}
          <p class="small">The map needs a layout library (about 150 MB, installed once).</p>
        {/if}
      {/if}
    </section>
  {/if}

  {#if problem}<p class="problem floating" role="alert">{problem}</p>{/if}
</main>

<style>
  /* The map is night whatever the system's look: dots glow on dark. Its own ink, then. */
  main {
    --night: #0e0f12;
    --ink-1: #f0efec;
    --ink-2: #c3c2b7;
    --ink-3: #8b8a83;
    --text-2: var(--ink-2);
    --other: #6b6e76;
    --panel: rgba(24, 25, 30, 0.82);
    --edge: rgba(255, 255, 255, 0.08);
    position: fixed;
    inset: 0;
    overflow: hidden;
    background: var(--night);
    color: var(--ink-1);
    font: 400 13px/1.45 var(--sans);
  }
  canvas {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    cursor: grab;
    touch-action: none;
  }
  canvas:active {
    cursor: grabbing;
  }
  canvas.pointing {
    cursor: pointer;
  }
  header {
    position: absolute;
    top: 16px;
    left: 16px;
    right: 16px;
    display: flex;
    gap: 10px;
    align-items: center;
    pointer-events: none;
  }
  header > * {
    pointer-events: auto;
  }
  form {
    display: flex;
    align-items: center;
    gap: 10px;
    width: min(420px, 100%);
    padding: 0 14px;
    height: 40px;
    border: 1px solid var(--edge);
    border-radius: 20px;
    background: var(--panel);
    backdrop-filter: blur(12px);
  }
  input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--ink-1);
    font: inherit;
    font-size: 14px;
  }
  input::placeholder {
    color: var(--ink-3);
  }
  button {
    font: inherit;
    color: var(--ink-2);
    cursor: pointer;
  }
  .toggle {
    display: flex;
    padding: 3px;
    border: 1px solid var(--edge);
    border-radius: 20px;
    background: var(--panel);
  }
  .toggle button {
    height: 32px;
    padding: 0 14px;
    border: 0;
    border-radius: 16px;
    background: transparent;
    font: 500 12px var(--mono);
    transition:
      background-color 180ms var(--ease),
      color 180ms var(--ease);
  }
  .toggle button.on {
    background: rgba(255, 255, 255, 0.12);
    color: var(--ink-1);
  }
  .quiet,
  .primary {
    height: 36px;
    padding: 0 16px;
    border: 1px solid var(--edge);
    border-radius: 18px;
    background: var(--panel);
    transition: transform 260ms var(--spring);
  }
  .quiet:active,
  .primary:active {
    transform: scale(0.96);
    transition-duration: 80ms;
  }
  .primary {
    margin-top: 6px;
    background: var(--ink-1);
    color: var(--night);
    font-weight: 500;
  }
  .legend {
    position: absolute;
    left: 16px;
    bottom: 16px;
    max-height: 50vh;
    overflow-y: auto;
    margin: 0;
    padding: 8px;
    list-style: none;
    border: 1px solid var(--edge);
    border-radius: 14px;
    background: var(--panel);
    backdrop-filter: blur(12px);
  }
  .legend button {
    display: flex;
    align-items: center;
    gap: 9px;
    width: 100%;
    padding: 5px 8px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    text-align: left;
  }
  .legend button:hover {
    background: rgba(255, 255, 255, 0.06);
  }
  .legend button.off {
    color: var(--ink-3);
  }
  .legend button.off .dot {
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
  .total {
    color: var(--ink-3);
    font: 400 11px var(--mono);
  }
  .total {
    padding: 6px 8px 2px;
  }
  .help {
    position: absolute;
    right: 16px;
    bottom: 12px;
    margin: 0;
    color: var(--ink-3);
    font-size: 11.5px;
    pointer-events: none;
  }
  .card {
    position: fixed;
    width: 320px;
    padding: 12px 14px;
    border: 1px solid var(--edge);
    border-radius: 12px;
    background: rgba(20, 21, 26, 0.94);
    pointer-events: none;
    animation: appear 140ms var(--ease) backwards;
  }
  @keyframes appear {
    from {
      opacity: 0;
      transform: translateY(3px);
    }
  }
  .card p {
    margin: 0;
  }
  .where {
    color: var(--ink-3);
    font: 400 11px var(--mono);
    text-transform: lowercase;
  }
  .file {
    margin-top: 2px !important;
    font-weight: 500;
    overflow-wrap: anywhere;
  }
  .text {
    margin-top: 8px !important;
    color: var(--ink-2);
    font-size: 12px;
    display: -webkit-box;
    -webkit-line-clamp: 6;
    line-clamp: 6;
    -webkit-box-orient: vertical;
    overflow: hidden;
    white-space: pre-line;
  }
  .hint {
    margin-top: 8px !important;
    color: var(--ink-3);
    font-size: 11px;
  }
  .marker {
    position: absolute;
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
    border: 2px solid #fff;
    border-radius: 50%;
    pointer-events: none;
    animation: pulse 1.6s var(--ease) infinite;
  }
  @keyframes pulse {
    0% {
      box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.5);
    }
    100% {
      box-shadow: 0 0 0 18px rgba(255, 255, 255, 0);
    }
  }
  .state {
    position: absolute;
    inset: 0;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 8px;
    padding: 24px;
    text-align: center;
  }
  .state h1 {
    margin: 0;
    font: 500 22px var(--display);
    letter-spacing: -0.01em;
  }
  .state p {
    max-width: 440px;
    margin: 0;
    color: var(--ink-2);
  }
  .small {
    font-size: 12px;
    color: var(--ink-3) !important;
  }
  .problem {
    color: #e66767 !important;
  }
  .floating {
    position: absolute;
    top: 68px;
    left: 16px;
    margin: 0;
    padding: 8px 12px;
    border-radius: 10px;
    background: var(--panel);
  }
  @media (prefers-reduced-motion: reduce) {
    .marker,
    .card {
      animation: none;
    }
  }
</style>
