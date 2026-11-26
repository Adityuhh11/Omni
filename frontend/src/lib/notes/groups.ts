/** The part of the list a note belongs under, by how long ago it was changed: what the history page's headings say. */
export function dayGroup(ms: number, now: Date = new Date()): string {
  const then = new Date(ms);
  const midnight = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((midnight(now) - midnight(then)) / 86_400_000);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 7) return "Earlier this week";
  if (days < 30) return "Earlier this month";
  return "Older";
}

/** The notes in the order given, in runs of the same group, each run with its heading. */
export function grouped<T extends { Modified: number }>(notes: T[], now: Date = new Date()): { heading: string; notes: T[] }[] {
  const runs: { heading: string; notes: T[] }[] = [];
  for (const note of notes) {
    const heading = dayGroup(note.Modified, now);
    if (runs.at(-1)?.heading === heading) runs.at(-1)!.notes.push(note);
    else runs.push({ heading, notes: [note] });
  }
  return runs;
}
