import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color: color,
    angle: rng.range(0, 360),
    spacing: rng.range(12, 32),
    thickness: rng.range(0.6, 2),
    sides: rng.integer(3, 8),
    rotation_step: rng.range(-12, 12),
  };
}
export type Parameters = ReturnType<typeof parameters>;
