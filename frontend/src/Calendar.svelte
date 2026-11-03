<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Window } from "@wailsio/runtime";
  import { Service as Calendars } from "../bindings/omni/internal/calendar";
  import type { Event as CalEvent, Status } from "../bindings/omni/internal/calendar";
  import Card from "./lib/Card.svelte";
  import RoundButton from "./lib/RoundButton.svelte";
  import "./lib/settings/forms.css";
  import { colorOf, covers, placeDay, sameDay, weekDays, weekStart } from "./lib/calendar";

  const HOUR = 46; // pixels an hour takes in the grid

  type Shown = CalEvent & { start: Date; end: Date; allDay: boolean; id: string };

  let start = $state(weekStart(new Date()));
  let events = $state<Shown[]>([]);
  let calendars = $state<Status[]>([]);
  let loaded = $state(false);
  let refreshing = $state(false);
  let now = $state(new Date());
  let open = $state<Shown | null>(null);
  let problem = $state("");
  let grid = $state<HTMLDivElement | undefined>();

  const days = $derived(weekDays(start));
  const title = $derived.by(() => {
    const first = days[0], last = days[6];
    const month = (d: Date, year: boolean) => d.toLocaleDateString(undefined, { month: "long", ...(year ? { year: "numeric" } : {}) });
    return first.getMonth() === last.getMonth() ? month(first, true) : `${month(first, first.getFullYear() !== last.getFullYear())} – ${month(last, true)}`;
  });
  const allDay = $derived(days.map((d) => events.filter((e) => e.allDay && covers(e, d))));
  const timed = $derived(days.map((d) => placeDay(events, d)));
  const thisWeek = $derived(days.some((d) => sameDay(d, now)));
  const failed = $derived(calendars.filter((c) => c.Error));

  async function load() {
    const list = (await Calendars.Between(start.getTime(), 7)) ?? [];
    events = list.map((e) => ({ ...e, id: e.ID, start: new Date(e.Start), end: new Date(e.End), allDay: e.AllDay }));
    calendars = (await Calendars.Calendars()) ?? [];
    loaded = true;
  }

  async function move(weeks: number) {
    open = null;
    start = weeks === 0 ? weekStart(new Date()) : new Date(start.getFullYear(), start.getMonth(), start.getDate() + 7 * weeks);
    await load();
    if (weeks === 0) scrollToNow();
  }

  async function refresh() {
    refreshing = true;
    try {
      await Calendars.Refresh();
      await load();
    } finally {
      refreshing = false; // the button comes back even if the call failed
    }
  }

  async function join(e: Shown) {
    problem = await Calendars.Join(e.Join);
  }

  async function scrollToNow() {
    await tick();
    const hour = thisWeek ? Math.max(0, now.getHours() - 1) : 8;
    grid?.scrollTo({ top: hour * HOUR });
  }

  const time = (d: Date) => d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  const span = (e: Shown) => {
    if (e.allDay) {
      const last = new Date(e.end.getTime() - 1);
      const day = (d: Date) => d.toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });
      return sameDay(e.start, last) ? `${day(e.start)}, all day` : `${day(e.start)} – ${day(last)}`;
    }
    return `${e.start.toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" })}, ${time(e.start)} – ${time(e.end)}`;
  };
  const hourLabel = (h: number) => new Date(2000, 0, 1, h).toLocaleTimeString(undefined, { hour: "numeric" });

  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape") {
      if (open) open = null;
      else Window.Close();
    } else if (e.key === "ArrowLeft" && !open) move(-1);
    else if (e.key === "ArrowRight" && !open) move(1);
    else if (e.key.toLowerCase() === "t" && !open) move(0);
  }

  onMount(() => {
    document.documentElement.dataset.surface = "card";
    load().then(scrollToNow);
    // The program fetches the calendars in the background; the window shows what it holds and keeps the time line moving.
    const timer = setInterval(() => {
      now = new Date();
      load();
    }, 60_000);
    return () => clearInterval(timer);
  });
</script>

<svelte:window onkeydown={onKey} />

<Card>
  <main class="cal">
    <header>
      <div class="drag" aria-hidden="true"></div>
      <h1>{title}</h1>
      <div class="nav">
        <button type="button" class="btn quiet" onclick={() => move(-1)} aria-label="Previous week" title="Previous week (←)">‹</button>
        <button type="button" class="btn" onclick={() => move(0)} disabled={thisWeek && sameDay(start, weekStart(now))} title="This week (T)">Today</button>
        <button type="button" class="btn quiet" onclick={() => move(1)} aria-label="Next week" title="Next week (→)">›</button>
      </div>
      <div class="end">
        {#if calendars.length}
          <ul class="legend" aria-label="Calendars">
            {#each calendars as c, i (c.ID)}
              <li title={c.Error || c.Name}><span class="dot" style:background={colorOf(i)}></span>{c.Name}</li>
            {/each}
          </ul>
          <button type="button" class="btn quiet" onclick={refresh} disabled={refreshing}>{refreshing ? "Refreshing…" : "Refresh"}</button>
        {/if}
        <RoundButton icon="close" label="Close" onclick={() => Window.Close()} />
      </div>
    </header>

    {#if loaded && calendars.length === 0}
      <div class="empty">
        <p class="big">No calendars yet</p>
        <p>Add Google, iCloud or Outlook by its subscription address. Omni reads it; nothing is changed.</p>
        <button type="button" class="btn primary" onclick={() => Calendars.OpenSettings()}>Add a calendar</button>
      </div>
    {:else}
      {#if failed.length}
        <p class="problem">{failed.map((c) => `${c.Name}: ${c.Error}`).join(" · ")}</p>
      {/if}
      {#if problem}<p class="problem">{problem}</p>{/if}

      <div class="days" role="row">
        <span class="gutter"></span>
        {#each days as d, i (d.getTime())}
          <div class="day" class:today={sameDay(d, now)} role="columnheader">
            <span class="weekday">{d.toLocaleDateString(undefined, { weekday: "short" })}</span>
            <span class="date">{d.getDate()}</span>
          </div>
          {#if i === 6}<span class="scrollpad"></span>{/if}
        {/each}
      </div>

      {#if allDay.some((list) => list.length)}
        <div class="allday">
          <span class="gutter label">all day</span>
          {#each allDay as list, i (i)}
            <div class="slot">
              {#each list as e (e.id)}
                <button type="button" class="chip" style:--c={colorOf(e.Color)} onclick={() => (open = e)}>{e.Title || "(no title)"}</button>
              {/each}
            </div>
          {/each}
          <span class="scrollpad"></span>
        </div>
      {/if}

      <div class="grid" bind:this={grid}>
        <div class="hours" style:height="{24 * HOUR}px">
          {#each Array.from({ length: 24 }, (_, h) => h) as h (h)}
            <span class="hour" style:top="{h * HOUR}px">{h === 0 ? "" : hourLabel(h)}</span>
          {/each}
        </div>
        {#each timed as list, i (days[i].getTime())}
          <div class="column" class:today={sameDay(days[i], now)} style:height="{24 * HOUR}px">
            {#each Array.from({ length: 24 }, (_, h) => h) as h (h)}<span class="line" style:top="{h * HOUR}px"></span>{/each}
            {#each list as p (p.event.id)}
              {@const e = p.event}
              <button
                type="button"
                class="event"
                class:short={p.to - p.from < 40}
                class:past={e.end < now}
                class:on={open?.id === e.id}
                style:--c={colorOf(e.Color)}
                style:top="{(p.from / 60) * HOUR + 1}px"
                style:height="{((p.to - p.from) / 60) * HOUR - 2}px"
                style:left="calc({(p.lane / p.lanes) * 100}% + 2px)"
                style:width="calc({100 / p.lanes}% - 4px)"
                onclick={() => (open = e)}
              >
                <span class="what">{e.Title || "(no title)"}</span>
                <span class="when">{time(e.start)}{e.Location ? ` · ${e.Location}` : ""}</span>
              </button>
            {/each}
            {#if sameDay(days[i], now)}
              <span class="now" style:top="{((now.getHours() * 60 + now.getMinutes()) / 60) * HOUR}px" aria-hidden="true"></span>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    {#if open}
      <div class="detail" role="dialog" aria-label={open.Title}>
        <div class="detail-head">
          <span class="dot" style:background={colorOf(open.Color)}></span>
          <h2>{open.Title || "(no title)"}</h2>
          <RoundButton icon="close" label="Close the event" size={28} onclick={() => (open = null)} />
        </div>
        <p class="meta">{span(open)}</p>
        <p class="meta">{open.Calendar}{open.Location ? ` · ${open.Location}` : ""}</p>
        {#if open.Join}
          <button type="button" class="btn primary join" onclick={() => open && join(open)}>Join the meeting</button>
        {/if}
        {#if open.Notes}<p class="notes">{open.Notes}</p>{/if}
      </div>
    {/if}
  </main>
</Card>

<style>
  .cal {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    color: var(--text);
    font-family: var(--sans);
  }
  header {
    position: relative;
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 14px 10px 24px;
  }
  .drag {
    position: absolute;
    inset: 0;
    --wails-draggable: drag;
  }
  header > :not(.drag) {
    position: relative; /* above the strip that drags, so they stay clickable */
  }
  h1 {
    margin: 0;
    min-width: 0;
    font: 620 18px/1.2 var(--sans);
    letter-spacing: -0.015em;
    white-space: nowrap;
  }
  .nav,
  .end {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .end {
    margin-left: auto;
    gap: 8px;
  }
  .nav .btn.quiet {
    width: 32px;
    padding: 0;
    font-size: 18px;
  }
  .legend {
    display: flex;
    gap: 12px;
    margin: 0 6px 0 0;
    padding: 0;
    list-style: none;
    color: var(--text-2);
    font-size: 12px;
  }
  .legend li {
    display: flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
  }
  .dot {
    flex: none;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .problem {
    margin: 0 24px 8px;
    color: var(--danger);
    font-size: 12px;
  }

  /* Columns: a gutter for the hours, seven days, and room for the grid's scrollbar so the headers line up with it. */
  .days,
  .allday {
    display: grid;
    grid-template-columns: 52px repeat(7, minmax(0, 1fr)) 10px;
    padding: 0 6px 0 0;
  }
  .grid {
    display: grid;
    grid-template-columns: 52px repeat(7, minmax(0, 1fr));
    flex: 1;
    min-height: 0;
    padding: 0 6px 0 0;
    overflow-y: scroll;
    border-top: 1px solid var(--line);
    scrollbar-gutter: stable;
  }
  .day {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 6px;
    padding: 6px 0 8px;
    color: var(--text-3);
  }
  .weekday {
    font: 560 11.5px/1 var(--sans);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .date {
    display: grid;
    place-items: center;
    min-width: 26px;
    height: 26px;
    border-radius: 13px;
    color: var(--text-2);
    font: 560 14px/1 var(--sans);
    font-variant-numeric: tabular-nums;
  }
  .day.today .weekday {
    color: var(--text);
  }
  .day.today .date {
    background: var(--chrome);
    color: var(--on-chrome);
    font-weight: 650;
  }

  .allday {
    padding-bottom: 6px;
  }
  .label {
    align-self: center;
    padding-right: 8px;
    color: var(--text-3);
    font-size: 10.5px;
    text-align: right;
  }
  .slot {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    padding: 0 2px;
  }
  .chip {
    min-width: 0;
    padding: 3px 7px;
    border: 0;
    border-radius: 6px;
    background: color-mix(in srgb, var(--c) 26%, transparent);
    color: var(--text);
    font: 540 11.5px/1.3 var(--sans);
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    cursor: pointer;
  }

  .hours {
    position: relative;
  }
  .hour {
    position: absolute;
    right: 8px;
    transform: translateY(-50%);
    color: var(--text-3);
    font: 500 10.5px/1 var(--sans);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .column {
    position: relative;
    border-left: 1px solid var(--line);
  }
  .column.today {
    background: color-mix(in srgb, var(--ink) 2.5%, transparent);
  }
  .line {
    position: absolute;
    right: 0;
    left: 0;
    border-top: 1px solid var(--line);
  }
  .event {
    position: absolute;
    display: flex;
    flex-direction: column;
    gap: 1px;
    box-sizing: border-box;
    min-width: 0;
    padding: 4px 6px 4px 8px;
    overflow: hidden;
    border: 0;
    border-radius: 7px;
    background: color-mix(in srgb, var(--c) 22%, var(--plate));
    box-shadow: inset 2px 0 0 var(--c);
    color: var(--text);
    text-align: left;
    cursor: pointer;
    transition: background-color 0.12s;
  }
  .event:hover,
  .event.on {
    background: color-mix(in srgb, var(--c) 34%, var(--plate));
  }
  .event.past {
    opacity: 0.55;
  }
  .event.short {
    flex-direction: row;
    gap: 6px;
    padding-top: 2px;
  }
  .what {
    overflow: hidden;
    font: 580 12px/1.25 var(--sans);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .when {
    overflow: hidden;
    color: var(--text-2);
    font: 500 11px/1.25 var(--sans);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .event:focus-visible,
  .chip:focus-visible {
    outline: 1px solid var(--chrome);
    outline-offset: 1px;
  }
  .now {
    position: absolute;
    right: 0;
    left: -1px;
    z-index: 1;
    height: 0;
    border-top: 1.5px solid var(--danger);
    pointer-events: none;
  }
  .now::before {
    content: "";
    position: absolute;
    top: -4.5px;
    left: -4px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--danger);
  }

  .empty {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 0 40px 40px;
    color: var(--text-2);
    font-size: 13px;
    text-align: center;
  }
  .empty p {
    margin: 0;
    max-width: 380px;
    line-height: 1.5;
  }
  .empty .big {
    color: var(--text);
    font: 620 16px/1.3 var(--sans);
  }
  .empty .btn {
    margin-top: 10px;
  }

  .detail {
    position: absolute;
    right: 18px;
    bottom: 18px;
    z-index: 3;
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 320px;
    max-height: calc(100% - 120px);
    padding: 14px 14px 16px 16px;
    overflow-y: auto;
    border: 1px solid var(--line);
    border-radius: 16px;
    background: var(--plate);
    box-shadow: var(--shadow), inset 0 1px 0 color-mix(in srgb, var(--ink) 12%, transparent);
  }
  .detail-head {
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }
  .detail-head .dot {
    margin-top: 7px;
  }
  .detail h2 {
    flex: 1;
    margin: 0;
    font: 620 15px/1.35 var(--sans);
  }
  .meta {
    margin: 0;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.45;
  }
  .join {
    align-self: flex-start;
    margin: 6px 0 2px;
  }
  .notes {
    margin: 6px 0 0;
    padding-top: 10px;
    border-top: 1px solid var(--line);
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.5;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    user-select: text;
  }
</style>
