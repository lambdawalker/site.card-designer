import type { Random } from "../../../random/seeded.ts";
export function parameters(rng: Random, color: string) {
  return {
    color,
    angle: rng.range(0, 180),
    num_lines: rng.integer(35, 100),
    step: 10,
    amplitude: rng.range(8, 30),
    scale: rng.range(0.002, 0.008),
    phase: rng.range(0, 1000),
    mod: rng.range(70, 160),
    noise_seed: rng.integer(0, 4294967295),
    thickness: rng.range(0.5, 2),
    width_noise_scale: 0.015,
    max_width: rng.range(2, 4),
  };
}
export type Parameters = ReturnType<typeof parameters>;
