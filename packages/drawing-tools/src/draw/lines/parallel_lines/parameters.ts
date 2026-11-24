import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color: color,
    angle: rng.range(0, 180),
    spacing: rng.range(12, 38),
    thickness: rng.range(0.5, 2),
    amplitude: rng.range(6, 30),
    wavelength: rng.range(45, 150),
    phase: rng.range(0, 100),
    noise_seed: rng.integer(0, 4294967295),
  };
}
export type Parameters = ReturnType<typeof parameters>;
