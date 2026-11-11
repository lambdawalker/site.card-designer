import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    start_color: "#ffffff",
    end_color: color,
    angle: rng.range(0, 360),
    wavelength: rng.range(80, 240),
  };
}
export type Parameters = ReturnType<typeof parameters>;
