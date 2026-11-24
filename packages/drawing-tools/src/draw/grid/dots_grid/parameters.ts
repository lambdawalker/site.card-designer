import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color: color,
    angle: rng.range(0, 360),
    radius: rng.range(1, 4),
    spacing: rng.range(12, 32),
  };
}
export type Parameters = ReturnType<typeof parameters>;
