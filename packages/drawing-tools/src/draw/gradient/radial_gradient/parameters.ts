import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    start_color: "#ffffff",
    end_color: color,
    center_x: rng.range(0.1, 0.9),
    center_y: rng.range(0.1, 0.9),
    radius: rng.range(0.6, 1.2),
  };
}
export type Parameters = ReturnType<typeof parameters>;
