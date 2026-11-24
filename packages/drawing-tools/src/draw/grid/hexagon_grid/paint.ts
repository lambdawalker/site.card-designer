import type { Parameters } from "./parameters.ts";
import { rotated, polygon } from "../../surface.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.fillStyle = p.color;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = p.thickness;
  rotated(ctx, p.angle, (extent) => {
    const step = p.radius * Math.sqrt(3);
    for (
      let row = -Math.ceil(extent / step);
      row <= Math.ceil(extent / step);
      row++
    ) {
      for (
        let col = -Math.ceil(extent / step);
        col <= Math.ceil(extent / step);
        col++
      ) {
        const x = col * step + ((row % 2) * step) / 2,
          y = row * p.radius * 1.5;
        polygon(ctx, x, y, p.radius, 6, Math.PI / 6);
        ctx.stroke();
      }
    }
  });
}
