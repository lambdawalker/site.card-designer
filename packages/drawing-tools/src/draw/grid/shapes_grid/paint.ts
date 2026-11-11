import type { Parameters } from "./parameters.ts";
import { rotated, polygon } from "../../surface.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.fillStyle = p.color;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = p.thickness;
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
        if (p.shape === "circle") {
          ctx.beginPath();
          ctx.arc(x, y, p.radius, 0, 2 * Math.PI);
        } else
          polygon(
            ctx,
            x,
            y,
            p.radius,
            p.shape === "square" ? 4 : p.shape === "triangle" ? 3 : p.sides,
            0,
            p.shape === "star",
          );
        ctx.fill();
        ctx.stroke();
      }
    }
  });
}
