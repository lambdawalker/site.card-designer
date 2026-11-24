import { countries, countryUrls } from "../../assets/countries.ts";
import type { Random } from "../../random/seeded.ts";
import { surface, context, dimensions } from "../surface.ts";
export function parameters(rng: Random, color: string) {
  return {
    asset: rng.pick(countries),
    color,
    stroke_color: color,
    blur: 2,
    thickness: 4,
  };
}
export type Parameters = ReturnType<typeof parameters>;
const images = new Map<string, Promise<HTMLImageElement>>();
async function load(asset: string) {
  if (!countries.includes(asset as (typeof countries)[number]))
    throw new Error("Unknown country asset");
  if (!images.has(asset)) {
    const task = (async () => {
      const image = new Image();
      image.src = countryUrls[asset];
      await image.decode();
      return image;
    })();
    images.set(asset, task);
    task.catch(() => images.delete(asset));
  }
  return images.get(asset)!;
}
export async function paint(ctx: CanvasRenderingContext2D, p: Parameters) {
  const image = await load(p.asset);
  const size = dimensions(ctx), scaleX = ctx.canvas.width / size.width, scaleY = ctx.canvas.height / size.height;
  const width = Math.round(size.width / 3.2), height = size.height;
  const maskSize = {width: width + 24, height: height + 24};
  const target = {width: Math.min(4096, Math.ceil(maskSize.width * scaleX)), height: Math.min(4096, Math.ceil(maskSize.height * scaleY))};
  const mask = surface(maskSize.width, maskSize.height, target),
    m = context(mask);
  const scale = Math.min(
    width / image.naturalWidth,
    height / image.naturalHeight,
  );
  const w = image.naturalWidth * scale,
    h = image.naturalHeight * scale;
  m.drawImage(image, 12 + (width - w) / 2, 12 + (height - h) / 2, w, h);
  m.globalCompositeOperation = "source-in";
  m.fillStyle = p.color;
  m.fillRect(0, 0, maskSize.width, maskSize.height);
  const outline = surface(maskSize.width, maskSize.height, target),
    o = context(outline);
  // Dilate the alpha mask before tinting to create an outer stroke.
  for (let a = 0; a < Math.PI * 2; a += Math.PI / 8)
    o.drawImage(
      mask,
      (Math.cos(a) * p.thickness) / 2,
      (Math.sin(a) * p.thickness) / 2,
      maskSize.width, maskSize.height,
    );
  o.globalCompositeOperation = "source-in";
  o.fillStyle = p.stroke_color;
  o.fillRect(0, 0, maskSize.width, maskSize.height);
  ctx.save();
  try {
    ctx.filter = "blur(" + p.blur * Math.max(scaleX, scaleY) + "px)";
    const x = size.width - size.width / 2.5 - 12,
      y = -12;
    ctx.drawImage(outline, x, y, maskSize.width, maskSize.height);
    ctx.drawImage(mask, x, y, maskSize.width, maskSize.height);
  } finally {
    ctx.restore();
  }
}
