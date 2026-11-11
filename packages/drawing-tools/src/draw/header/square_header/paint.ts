import { dimensions } from "../../surface.ts";
import type { Parameters } from "./parameters.ts";

export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  const { width, height } = dimensions(ctx),
    h = height * p.height;
  ctx.fillStyle = p.color;
  ctx.beginPath();
  ctx.moveTo(0, 0);
  ctx.lineTo(width, 0);
  ctx.lineTo(width, h);
  ctx.lineTo(0, h);
  ctx.closePath();
  ctx.fill();
}
