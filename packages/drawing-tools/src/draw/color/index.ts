import { Hsluv } from "hsluv";
import type { Random } from "../../random/seeded.ts";
export function rgb(hex: string): [number, number, number] {
  return [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255) as [
    number,
    number,
    number,
  ];
}
export function related(hex: string, rng: Random, alpha: number): string {
  const c = new Hsluv();
  [c.rgb_r, c.rgb_g, c.rgb_b] = rgb(hex);
  c.rgbToHsluv();
  c.hsluv_h = (c.hsluv_h + rng.range(-25, 25) + 360) % 360;
  c.hsluv_l = Math.min(95, Math.max(15, c.hsluv_l + rng.range(-24, 12)));
  c.hsluvToRgb();
  return `rgba(${Math.round(c.rgb_r * 255)},${Math.round(c.rgb_g * 255)},${Math.round(c.rgb_b * 255)},${alpha})`;
}
