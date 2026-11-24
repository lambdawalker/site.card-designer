import type { Parameters } from "./parameters.ts";
import { rotated, polygon } from "../../surface.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.fillStyle = p.color;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = 1;
  rotated(ctx, p.angle, (extent) => {
    const step = p.radius * 2 + p.spacing;
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
        const x = col * step,
          y = row * step;
        ctx.beginPath();
        ctx.arc(x, y, p.radius, 0, Math.PI * 2);
        ctx.fill();
      }
    }
  });
}
