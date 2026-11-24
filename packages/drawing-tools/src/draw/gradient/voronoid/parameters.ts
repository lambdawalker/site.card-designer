import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color_a: "#ffffff",
    color_b: color,
    timestamp: rng.range(0, 20),
    scale: rng.range(5, 30),
    angle_degrees: rng.range(0, 360),
  };
}
export type Parameters = ReturnType<typeof parameters>;
