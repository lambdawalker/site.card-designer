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
  for (let i = 100; i >= 0; i--) {
    const x = (i / 100) * width;
    ctx.lineTo(
      x,
      h + p.depth * Math.sin((2 * Math.PI * p.frequency * i) / 100),
    );
  }
  ctx.closePath();
  ctx.fill();
}
