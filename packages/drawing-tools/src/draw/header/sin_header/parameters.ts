import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color: color,
    height: rng.range(0.12, 0.3),
    depth: rng.range(10, 45),
    frequency: rng.range(0.5, 2.5),
  };
}
export type Parameters = ReturnType<typeof parameters>;
