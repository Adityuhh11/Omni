<script lang="ts">
  import type { Source } from "../../../bindings/omni/internal/chat";
  import { Service as Calendars } from "../../../bindings/omni/internal/calendar";
  import Icon from "../Icon.svelte";
  import { colorOf, meetingName, sameDay } from "../calendar";

  // The calendar events an answer drew on, by day: the answer says what they mean, this shows them as they are.
  let { events }: { events: Source[] } = $props();

  const now = new Date();
  const days = $derived.by(() => {
    const out: { day: Date; items: Source[] }[] = [];
    for (const s of events) {
      const start = new Date(s.event!.start);
      const last = out.at(-1);
      if (last && sameDay(last.day, start)) last.items.push(s);
      else out.push({ day: start, items: [s] });
    }
    return out;
  });

  const dayName = (d: Date) => {
    const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
    if (sameDay(d, now)) return "Today";
    if (sameDay(d, tomorrow)) return "Tomorrow";
    return d.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "short" });
  };
  const time = (ms: number) => new Date(ms).toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
</script>

<div class="agenda" aria-label="Calendar events">
  {#each days as group (group.day.getTime())}
    <p class="day">{dayName(group.day)}</p>
    <ul>
      {#each group.items as s (s.event!.id)}
        {@const e = s.event!}
        <li class:past={e.end < now.getTime()} style:--c={colorOf(e.color)}>
          <span class="when">{e.all_day ? "All day" : `${time(e.start)} – ${time(e.end)}`}</span>
          <span class="what"><span class="title">{s.title || "(no title)"}</span><span class="cal">{e.calendar}</span></span>
          {#if e.join && e.end > now.getTime()}
            <button type="button" class="join" title="Join {meetingName(e.join)}" onclick={() => Calendars.Join(e.join ?? "")}>
              <Icon name="camera" size={14} />Join
            </button>
          {/if}
        </li>
      {/each}
    </ul>
  {/each}
</div>

<style>
  .agenda {
    margin-top: 10px;
    padding: 6px 12px 8px;
    border: 1px solid var(--line);
    border-radius: 12px;
    background: color-mix(in srgb, var(--ink) 2.5%, transparent);
  }
  .day {
    margin: 8px 0 4px;
    color: var(--text-3);
    font: 560 11.5px/1.2 var(--sans);
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 34px;
    padding: 3px 0 3px 10px;
    box-shadow: inset 2px 0 0 var(--c);
  }
  li + li {
    margin-top: 2px;
  }
  li.past {
    opacity: 0.55;
  }
  .when {
    flex: none;
    width: 118px;
    color: var(--text-2);
    font: 500 12px/1.3 var(--sans);
    font-variant-numeric: tabular-nums;
  }
  .what {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }
  .title,
  .cal {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .title {
    color: var(--text);
    font: 560 13px/1.3 var(--sans);
  }
  .cal {
    color: var(--text-3);
    font-size: 11.5px;
  }
  .join {
    display: inline-flex;
    flex: none;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 10px;
    border: 1px solid color-mix(in srgb, var(--c) 45%, transparent);
    border-radius: 13px;
    background: color-mix(in srgb, var(--c) 16%, transparent);
    color: var(--text);
    font: 560 12px/1 var(--sans);
    cursor: pointer;
  }
  .join:hover {
    background: color-mix(in srgb, var(--c) 28%, transparent);
  }
  .join:focus-visible {
    outline: 1px solid var(--chrome);
    outline-offset: 2px;
  }
</style>
