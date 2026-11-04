/** The week view's arithmetic: which days a week has, which part of an event falls on a day, and how events that overlap share
 *  the width of the day. The page only draws what these return. */

export type Span = { id: string; start: Date; end: Date; allDay: boolean };

/** A placed event: its part of the day in minutes from midnight, and its lane of lanes (side by side when they overlap). */
export type Placed<T> = { event: T; from: number; to: number; lane: number; lanes: number };

const DAY = 24 * 60;

/** Midnight at the start of the Monday of the week that holds day. ponytail: weeks start on Monday everywhere. */
export function weekStart(day: Date): Date {
  const d = new Date(day.getFullYear(), day.getMonth(), day.getDate());
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7));
  return d;
}

/** The seven midnights of the week that starts at start. */
export function weekDays(start: Date): Date[] {
  return Array.from({ length: 7 }, (_, i) => new Date(start.getFullYear(), start.getMonth(), start.getDate() + i));
}

export function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

/** Whether an all-day event (or one that spans whole days) is on day. */
export function covers(e: Span, day: Date): boolean {
  const next = new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1);
  return e.start < next && e.end > day;
}

/** The timed events of a day, each cut to the day (one that runs past midnight shows on both days) and given a lane. Events that
 *  overlap, directly or through others, share the day's width equally; a short event still gets a usable height (minMinutes). */
export function placeDay<T extends Span>(events: T[], day: Date, minMinutes = 20): Placed<T>[] {
  // Positions are the time the clock shows, not time elapsed since midnight: on the days the clocks change a day is 23 or 25
  // hours long, and an event at 15:00 still belongs at the 15:00 line.
  const next = new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1);
  const clock = (d: Date) => d.getHours() * 60 + d.getMinutes() + d.getSeconds() / 60;
  const parts = events
    .filter((e) => !e.allDay && e.start < next && (e.end > day || (e.end.getTime() === e.start.getTime() && e.start >= day)))
    .map((e) => ({ event: e, from: e.start < day ? 0 : clock(e.start), to: e.end >= next ? DAY : clock(e.end) }))
    .map((p) => ({ ...p, to: Math.max(p.to, p.from + minMinutes) }))
    .sort((a, b) => a.from - b.from || b.to - a.to || a.event.id.localeCompare(b.event.id));

  const placed: Placed<T>[] = [];
  let cluster: Placed<T>[] = [];
  let laneEnds: number[] = [];
  let clusterEnd = -1;
  const close = () => {
    for (const p of cluster) p.lanes = laneEnds.length;
    cluster = [];
    laneEnds = [];
  };
  for (const p of parts) {
    if (p.from >= clusterEnd) close();
    let lane = laneEnds.findIndex((end) => end <= p.from);
    if (lane < 0) lane = laneEnds.push(0) - 1;
    laneEnds[lane] = p.to;
    const item = { ...p, lane, lanes: 1 };
    cluster.push(item);
    placed.push(item);
    clusterEnd = Math.max(clusterEnd, p.to);
  }
  close();
  return placed;
}

/** The colour of the nth calendar: muted metals, the same family as the ring, so the plate stays black and chrome. */
export const CALENDAR_COLORS = ["#8fb8d8", "#c98a5e", "#6fbf95", "#a690d8", "#d88fa6", "#cdb06a"];

export const colorOf = (n: number) => CALENDAR_COLORS[((n % CALENDAR_COLORS.length) + CALENDAR_COLORS.length) % CALENDAR_COLORS.length];
