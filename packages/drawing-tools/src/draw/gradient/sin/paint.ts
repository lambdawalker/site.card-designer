import type { Parameters } from "./parameters.ts";
import { rgb } from "../../color/index.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  const { width, height } = ctx.canvas,
    scaleX = ctx.getTransform().a, scaleY = ctx.getTransform().d,
    a = (p.angle * Math.PI) / 180;
  const start = rgb(p.start_color),
    end = rgb(p.end_color),
    pixels = ctx.createImageData(width, height);
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++) {
      const mix =
          (Math.cos(
            (((x / scaleX) * Math.cos(a) + (y / scaleY) * Math.sin(a)) * 2 * Math.PI) / p.wavelength,
          ) +
            1) /
          2,
        i = (y * width + x) * 4;
      for (let c = 0; c < 3; c++)
        pixels.data[i + c] = Math.round(
          (start[c] * (1 - mix) + end[c] * mix) * 255,
        );
      pixels.data[i + 3] = 255;
    }
  ctx.putImageData(pixels, 0, 0);
}
