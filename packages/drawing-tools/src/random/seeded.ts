/** Versioned LCG. Streams are derived from labels, never from render order. */
export function random(seed: number, label = "") {
  let state = seed >>> 0;
  for (const char of label)
    state = Math.imul(state ^ char.charCodeAt(0), 16777619) >>> 0;
  const next = () => {
    state = (Math.imul(1664525, state) + 1013904223) >>> 0;
    return state / 4294967296;
  };
  return {
    next,
    range: (low: number, high: number) => low + next() * (high - low),
    integer: (low: number, high: number) =>
      low + Math.floor(next() * (high - low + 1)),
    pick: <T>(values: readonly T[]): T =>
      values[Math.floor(next() * values.length)],
  };
}
export type Random = ReturnType<typeof random>;
