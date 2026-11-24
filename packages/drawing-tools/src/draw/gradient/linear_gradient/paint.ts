import { dimensions } from "../../surface.ts";
import type { Parameters } from "./parameters.ts";

export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  const { width, height } = dimensions(ctx),
    a = (p.angle * Math.PI) / 180,
    d = Math.abs(width * Math.cos(a)) + Math.abs(height * Math.sin(a));
  const dx = (Math.cos(a) * d) / 2,
    dy = (Math.sin(a) * d) / 2;
  const g = ctx.createLinearGradient(
    width / 2 - dx,
    height / 2 - dy,
    width / 2 + dx,
    height / 2 + dy,
  );
  g.addColorStop(0, p.start_color);
  g.addColorStop(1, p.end_color);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, width, height);
}
