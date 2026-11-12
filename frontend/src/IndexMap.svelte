<script lang="ts">
  import { onMount } from "svelte";
  import { Service as Atlas } from "../bindings/omni/internal/atlas";
  import type { View } from "../bindings/omni/internal/atlas";
  import type { MapPoint, MapTopic } from "../bindings/omni/internal/sidecar";
  import Loader from "./lib/Loader.svelte";
  import { createRenderer, type Renderer } from "./lib/map/gl";
  import { ageColors, colorsFor, folderName, slotColor } from "./lib/map/folders";
  import { piecesOf } from "./lib/map/files";
  import Legend from "./lib/map/Legend.svelte";
  import {
    approach,
    clamp,
    decodeFloats,
    decodeUint16,
    decodeUint32,
    FOV,
    MAX_DISTANCE,
    MAX_PITCH,
    MIN_DISTANCE,
    pick,
    toScreen,
    viewProjection,
    type Camera,
    type Mat4,
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

  // The map itself, and what is known of each point.
  type Mode = "folder" | "type" | "age";
  let coords: Float32Array | null = null;
  let source: Uint16Array | null = null;
  let fileOf: Uint32Array | null = null;
  let rankOf: Uint32Array | null = null;
  let typeOf: Uint16Array | null = null;
  let modified: Uint32Array | null = null;
  let roots = $state<string[]>([]);
  let counts = $state<number[]>([]);
  let types = $state<string[]>([]);
  let typeCounts = $state<number[]>([]);
  let topics = $state<MapTopic[]>([]);

  // How it is coloured; a group hidden from the legend stays hidden only while its colouring is on.
  let mode = $state<Mode>("folder");
  let hiddenBy = $state({ folder: new Set<number>(), type: new Set<number>() });
  let cuts = $state<[number, number, number] | null>(null);
  let showLabels = $state(true);
  let labels = $state<{ text: string; x: number; y: number; size: number; fade: number }[]>([]);

  // The camera eases toward the goal every frame; input moves the goal.
  const now: Camera = structuredClone(HOME);
  const goal: Camera = structuredClone(HOME);
  let flatView = $state(false);
  let lastInput = performance.now();

  // Hover and query.
  let hover = $state<{ index: number; x: number; y: number; point: MapPoint | null } | null>(null);
  let query = $state("");
  let located = $state<{ at: [number, number, number]; points: number[] } | null>(null);
  // A clicked dot: every piece of its file lights up, joined in reading order.
  let selected = $state<{ index: number; pieces: number; point: MapPoint | null } | null>(null);
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
      fileOf = decodeUint32(points.file ?? "");
      rankOf = decodeUint32(points.rank ?? "");
      typeOf = decodeUint16(points.type ?? "");
      modified = decodeUint32(points.modified ?? "");
      roots = points.roots ?? [];
      types = points.types ?? [];
      topics = points.topics ?? [];
      counts = tally(source, roots.length);
      typeCounts = tally(typeOf, types.length);
      renderer?.setPoints(coords, colors());
      kick();
    } catch (error) {
      problem = "The map could not be read: " + String(error);
    }
  }

  function tally(group: Uint16Array, size: number): number[] {
    const out = new Array<number>(size).fill(0);
    for (const g of group) out[g]++;
    return out;
  }

  function colors(): Float32Array {
    if (mode === "age") {
      const age = ageColors(modified!);
      cuts = age.cuts;
      return age.colors;
    }
    return mode === "type" ? colorsFor(typeOf!, types, hiddenBy.type) : colorsFor(source!, roots, hiddenBy.folder);
  }

  function visible(i: number): boolean {
    if (mode === "folder") return !hiddenBy.folder.has(source![i]);
    if (mode === "type") return !hiddenBy.type.has(typeOf![i]);
    return true;
  }

  async function build() {
    coords = null;
    located = null;
    selected = null;
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
    const dimGoal = located || selected ? 0.22 : 1;
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
    labels = showLabels ? placeLabels(m, dpr) : [];
  }

  // A region's name sits over its middle, larger for a larger region, fading out toward the far side of the map.
  function placeLabels(m: Mat4, dpr: number) {
    const biggest = topics[0]?.count ?? 1;
    return topics.flatMap((t) => {
      const [x, y, z] = t.at ?? [0, 0, 0];
      const at = toScreen(m, x, y, z * (1 - now.flat), canvas.width, canvas.height);
      if (!at) return [];
      const fade = clamp((now.distance / at[2] - 0.55) / 0.35, 0, 1);
      return fade > 0 ? [{ text: t.label, x: at[0] / dpr, y: at[1] / dpr, size: 11 + 4 * Math.sqrt(t.count / biggest), fade }] : [];
    });
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
    if (!clicked) return;
    if (hover) select(hover.index);
    else clearSelection();
  }

  function light(points: ArrayLike<number>) {
    if (!coords) return;
    const lit = new Float32Array(coords.length / 3);
    for (let i = 0; i < points.length; i++) lit[points[i]] = 1;
    renderer?.setLit(lit);
  }

  function select(index: number) {
    if (!fileOf || !rankOf) return;
    const pieces = piecesOf(fileOf, rankOf, fileOf[index]);
    located = null;
    light(pieces);
    renderer?.setPath(pieces);
    selected = { index, pieces: pieces.length, point: hover?.index === index ? hover.point : null };
    if (!selected.point) {
      Atlas.Point(index)
        .then((point) => {
          if (selected?.index === index) selected = { ...selected, point };
        })
        .catch(() => {});
    }
    touched();
  }

  function clearSelection() {
    if (!selected) return;
    selected = null;
    renderer?.setPath(new Uint32Array());
    light(located?.points ?? []);
    touched();
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
    const index = pick(matrix(), coords, now.flat, visible, (clientX - rect.left) * dpr,
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

  function toggleGroup(index: number) {
    if (mode === "age") return;
    const next = new Set(hiddenBy[mode]);
    if (!next.delete(index)) next.add(index);
    hiddenBy = { ...hiddenBy, [mode]: next };
    if (coords) renderer?.setColors(colors());
    kick();
  }

  function setMode(next: Mode) {
    mode = next;
    if (coords) renderer?.setColors(colors());
    hover = null;
    kick();
  }

  function toggleLabels() {
    showLabels = !showLabels;
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
      selected = null;
      renderer?.setPath(new Uint32Array());
      located = { at, points: found.points ?? [] };
      light(located.points);
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
    if (!located) return;
    located = null;
    light([]);
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
    if (e.key === "Escape") {
      clearQuery();
      clearSelection();
    } else if (e.key === "f" || e.key === "F") home();
    else if (e.key === "l" || e.key === "L") toggleLabels();
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
  const entries = $derived(
    mode === "type"
      ? types.map((t, i) => ({ name: t === "other" ? "Other types" : t, title: t, color: slotColor(i, t), count: typeCounts[i] ?? 0 }))
      : roots.map((r, i) => ({ name: folderName(r), title: r || "A folder no longer searched", color: slotColor(i, r), count: counts[i] ?? 0 })),
  );
  const MODES: [Mode, string][] = [
    ["folder", "Folder"],
    ["type", "Type"],
    ["age", "Age"],
  ];
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

  {#each labels as label (label.text)}
    <span class="topic" style:left="{label.x}px" style:top="{label.y}px" style:font-size="{label.size}px"
      style:opacity={label.fade * (located || selected ? 0.35 : 1)} aria-hidden="true">{label.text}</span>
  {/each}

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
      <div class="toggle" role="group" aria-label="Colour by">
        {#each MODES as [id, text]}
          <button class:on={mode === id} onclick={() => setMode(id)} aria-pressed={mode === id}>{text}</button>
        {/each}
      </div>
      {#if topics.length}
        <div class="toggle">
          <button class:on={showLabels} onclick={toggleLabels} aria-pressed={showLabels}>Labels</button>
        </div>
      {/if}
      {#if view?.Stale}
        <button class="quiet" onclick={build} title="The index has changed since this map was made">Rebuild</button>
      {/if}
    </header>

    <Legend label={mode === "type" ? "File types" : mode === "age" ? "Age" : "Folders"} {entries}
      hidden={mode === "age" ? new Set() : hiddenBy[mode]} cuts={mode === "age" ? cuts : null} {total} ontoggle={toggleGroup} />

    <p class="help">Drag to turn · right-drag to move · scroll to zoom · click a dot for its whole file · double-click to fly in · L for labels · F to come back</p>
  {/if}

  {#if hover?.point}
    {@const card = hover.point}
    <aside class="card" style:left="{Math.min(hover.x + 16, innerWidth - 336)}px" style:top="{Math.min(hover.y + 16, innerHeight - 200)}px">
      <p class="where">{folderName(card.root)} · {card.kind.replace("_", " ")}</p>
      <p class="file">{card.uri}</p>
      {#if card.text}<p class="text">{card.text}</p>{/if}
      <p class="hint">Click to show the whole file</p>
    </aside>
  {/if}

  {#if selected}
    {@const index = selected.index}
    <aside class="card pinned" aria-live="polite">
      <p class="where">
        {selected.point ? `${folderName(selected.point.root)} · ` : ""}{selected.pieces === 1 ? "1 piece" : `${selected.pieces.toLocaleString()} pieces`}
      </p>
      <p class="file">{selected.point?.uri ?? "…"}</p>
      {#if selected.pieces > 1}<p class="hint">The line runs through its pieces in reading order.</p>{/if}
      <div class="actions">
        <button class="quiet" onclick={() => Atlas.Open(index).catch((error) => (problem = String(error)))}>Open the file</button>
        <button class="quiet" onclick={clearSelection}>Close</button>
      </div>
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
  /* The map is night whatever the system's look. Its own ink, then. */
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
  .topic {
    position: absolute;
    transform: translate(-50%, -50%);
    color: var(--ink-1);
    font-family: var(--sans);
    font-weight: 500;
    white-space: nowrap;
    text-shadow:
      0 0 3px var(--night),
      0 0 8px var(--night),
      0 1px 2px var(--night);
    pointer-events: none;
    transition: opacity 180ms var(--ease);
  }
  .card.pinned {
    top: 68px;
    right: 16px;
    pointer-events: auto;
  }
  .actions {
    display: flex;
    gap: 8px;
    margin-top: 10px;
  }
  .actions .quiet {
    height: 30px;
    padding: 0 12px;
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
