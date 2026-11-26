// The history window and the note window are two pages with one storage. When history deletes notes, the note window may hold one of
// them open, and its next autosave would write it back: so history says what it deleted, through the storage event the other page hears.

const KEY = "omni-notes-deleted";

/** Tells the other windows that these notes are gone. */
export function announceGone(ids: string[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify({ ids, at: Date.now() })); // the time makes every announcement a change, even of the same notes
  } catch {
    // no storage in this window: the note window finds out when it next tries to open the note
  }
}

/** Calls `fn` with the ids of notes another window deleted; returns what stops it. */
export function onGone(fn: (ids: string[]) => void): () => void {
  const heard = (event: StorageEvent) => {
    if (event.key !== KEY || !event.newValue) return;
    try {
      fn((JSON.parse(event.newValue) as { ids: string[] }).ids);
    } catch {
      // not what this window wrote: ignored
    }
  };
  addEventListener("storage", heard);
  return () => removeEventListener("storage", heard);
}
