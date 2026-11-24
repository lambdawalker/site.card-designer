import type { Parameters } from "./parameters.ts";
import { rotated, polygon } from "../../surface.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  ctx.fillStyle = p.color;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = p.thickness;
  rotated(ctx, p.angle, (extent) => {
    for (let r = p.spacing; r < extent; r += p.spacing) {
      polygon(
        ctx,
        0,
        0,
        r,
        p.sides,
        ((r / p.spacing) * p.rotation_step * Math.PI) / 180,
      );
      ctx.stroke();
    }
  });
}
