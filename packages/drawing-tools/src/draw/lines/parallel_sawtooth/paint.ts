import type { Parameters } from "./parameters.ts";
import { rotated } from "../../surface.ts";
import { noise } from "../../../random/noise.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.strokeStyle = p.color;
  ctx.fillStyle = p.color;
  ctx.lineWidth = p.thickness;
  rotated(ctx, p.angle, (extent) => {
    for (let y = -extent; y <= extent; y += p.spacing) {
      ctx.beginPath();
      for (let x = -extent; x <= extent; x += 2) {
        const dy = (((((x / p.wavelength) % 1) + 1) % 1) * 2 - 1) * p.amplitude;
        if (x === -extent) ctx.moveTo(x, y + dy);
        else ctx.lineTo(x, y + dy);
      }
      ctx.stroke();
    }
  });
}
