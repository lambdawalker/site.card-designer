import type { Parameters } from "./parameters.ts";
import { rotated, polygon } from "../../surface.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.fillStyle = p.color;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = p.thickness;
  rotated(ctx, p.angle, (extent) => {
    const step = p.radius;
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
        const x = (col * step) / 2,
          y = (row * step * Math.sqrt(3)) / 2;
        ctx.beginPath();
        ctx.moveTo(x, y);
        ctx.lineTo(x + step / 2, y + (step * Math.sqrt(3)) / 2);
        ctx.lineTo(x - step / 2, y + (step * Math.sqrt(3)) / 2);
        ctx.closePath();
        ctx.stroke();
      }
    }
  });
}
