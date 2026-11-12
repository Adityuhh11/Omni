/** The points of one file, in reading order: what lights up, joined by a line, when a dot is clicked. */
export function piecesOf(file: Uint32Array, rank: Uint32Array, which: number): Uint32Array {
  const found: number[] = [];
  for (let i = 0; i < file.length; i++) if (file[i] === which) found.push(i);
  found.sort((a, b) => rank[a] - rank[b]);
  return Uint32Array.from(found);
}
