export interface Size {
  width: number;
  height: number;
}
const logicalSizes = new WeakMap<HTMLCanvasElement, Size>();
export function dimensions(ctx: CanvasRenderingContext2D): Size {
  return logicalSizes.get(ctx.canvas) || ctx.canvas;
}
export function surface(width: number, height: number, target?: Size): HTMLCanvasElement {
  if (
    !Number.isInteger(width) ||
    !Number.isInteger(height) ||
    width < 1 ||
    height < 1 ||
    width > 4096 ||
    height > 4096
  )
    throw new Error("Canvas dimensions must be integers from 1 to 4096");
  if (target && (!Number.isInteger(target.width) || !Number.isInteger(target.height) || target.width < 1 || target.height < 1 || target.width > 4096 || target.height > 4096))
    throw new Error("Render dimensions must be integers from 1 to 4096");
  const canvas = document.createElement("canvas");
  canvas.width = target?.width ?? width;
  canvas.height = target?.height ?? height;
  logicalSizes.set(canvas, {width, height});
  return canvas;
}
export function context(canvas: HTMLCanvasElement): CanvasRenderingContext2D {
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Canvas 2D unavailable");
  const size = dimensions(ctx);
  ctx.setTransform(canvas.width / size.width, 0, 0, canvas.height / size.height, 0, 0);
  return ctx;
}
export function rotated(
  ctx: CanvasRenderingContext2D,
  angle: number,
  draw: (extent: number) => void,
) {
  ctx.save();
  try {
    ctx.translate(dimensions(ctx).width / 2, dimensions(ctx).height / 2);
    ctx.rotate((angle * Math.PI) / 180);
    draw(Math.hypot(dimensions(ctx).width, dimensions(ctx).height));
  } finally {
    ctx.restore();
  }
}
export function polygon(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  sides: number,
  angle = 0,
  star = false,
) {
  ctx.beginPath();
  const count = star ? sides * 2 : sides;
  for (let i = 0; i < count; i++) {
    const a = angle + (i * 2 * Math.PI) / count,
      r = star && i % 2 ? radius * 0.45 : radius;
    const px = x + r * Math.cos(a),
      py = y + r * Math.sin(a);
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.closePath();
}
