import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color: color,
    angle: rng.range(0, 360),
    radius: rng.range(8, 28),
    thickness: rng.range(0.6, 2),
  };
}
export type Parameters = ReturnType<typeof parameters>;
