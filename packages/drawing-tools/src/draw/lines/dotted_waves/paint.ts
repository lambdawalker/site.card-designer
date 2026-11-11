import type { Parameters } from "./parameters.ts";
import { rotated } from "../../surface.ts";
import { noise } from "../../../random/noise.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters) {
  ctx.strokeStyle = p.color;
  ctx.fillStyle = p.color;
  rotated(ctx, p.angle, (extent) => {
    for (let i = 0; i < p.num_lines; i++) {
      let x = -extent / 2,
        y = -extent / 2 + (extent * i) / p.num_lines;
      ctx.beginPath();
      ctx.moveTo(x, y);
      for (let s = 0; s < Math.ceil(extent / p.step) + 2; s++) {
        const variation =
          (noise((x + p.phase) * p.scale, y * p.scale, p.noise_seed) +
            noise((x + p.phase) * p.scale * 2, y * p.scale * 2, p.noise_seed) *
              0.5) /
          1.5;
        x += p.step;
        y += variation * p.amplitude * Math.sin((x + y) / p.mod);

        const radius = Math.max(
          0.5,
          (p.thickness +
            ((p.max_width - p.thickness) *
              (noise(s * p.step * p.width_noise_scale, i * 0.2, p.noise_seed) +
                1)) /
              2) /
            2,
        );
        ctx.lineWidth = radius * 2;
        ctx.lineCap = "round";
        ctx.lineTo(x, y);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(x, y);
      }
      ctx.lineWidth = p.thickness;
      ctx.stroke();
    }
  });
}
