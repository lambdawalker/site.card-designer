import { dimensions } from "../../surface.ts";
import type { Parameters } from "./parameters.ts";

export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  const { width, height } = dimensions(ctx);
  const g = ctx.createRadialGradient(
    p.center_x * width,
    p.center_y * height,
    0,
    p.center_x * width,
    p.center_y * height,
    p.radius * Math.hypot(width, height),
  );
  g.addColorStop(0, p.start_color);
  g.addColorStop(1, p.end_color);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, width, height);
}
