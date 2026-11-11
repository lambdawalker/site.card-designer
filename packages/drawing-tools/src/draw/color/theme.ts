import { Hsluv } from "hsluv";
import { random } from "../../random/seeded.ts";

export interface Theme {
  primary_color: string;
  text_color: "#000000";
  hue: number;
  saturation: number;
  lightness: number;
}

/** Seeded HSLuv colors using the original generator's inclusive ranges. */
export function generateTheme(seed: number): Theme {
  const rng = random(seed, "theme-v1");
  const hue = rng.integer(0, 360);
  const saturation = rng.integer(30, 60);
  let lightness = rng.integer(30, 80);
  const color = new Hsluv();
  color.hsluv_h = hue;
  color.hsluv_s = saturation;
  let primary_color: string;
  // Keep the original black text while lifting dark samples to AA contrast.
  // Measure rounded output hex values, as those are what will be rendered.
  for (;;) {
    color.hsluv_l = lightness;
    color.hsluvToHex();
    primary_color = color.hex.toLowerCase();
    const channels = [1, 3, 5].map((offset) => {
      const channel = parseInt(primary_color.slice(offset, offset + 2), 16) / 255;
      return channel <= 0.04045
        ? channel / 12.92
        : ((channel + 0.055) / 1.055) ** 2.4;
    });
    const luminance =
      channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
    if ((luminance + 0.05) / 0.05 >= 4.5 || lightness === 80) break;
    lightness++;
  }
  return {
    primary_color,
    text_color: "#000000",
    hue,
    saturation,
    lightness,
  };
}
