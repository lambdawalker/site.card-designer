import { surface, context } from "../../surface.ts";
import { paint } from "./paint.ts";
import type { Parameters } from "./parameters.ts";
export function create(width: number, height: number, p: Parameters) {
  const canvas = surface(width, height);
  paint(context(canvas), p);
  return canvas;
}
