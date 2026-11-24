import type {HeaderLayout} from '../header/layout.ts';
import { random } from "../../random/seeded.ts";
import { related } from "../color/index.ts";
import { registry, groups } from "../registry.ts";
import type { Layer, ComponentId } from "../registry.ts";
export const version = "0.1.0";
export type Family = "type-a" | "type-b" | "type-c" | "type-d";
export interface Options {
  family: Family;
  seed: number;
  color: string;
  width: number;
  height: number;
}
export interface Recipe extends Options {
  version: string;
  layers: Layer[];
  header_layout?: HeaderLayout;
}
export function parameters(options: Options): Recipe {
  const { width, height, seed, color, family } = options;
  if (
    !Number.isInteger(width) ||
    !Number.isInteger(height) ||
    width < 1 ||
    height < 1 ||
    width > 4096 ||
    height > 4096
  )
    throw new Error("Dimensions must be integers from 1 to 4096");
  if (!Number.isInteger(seed) || seed < 0 || seed > 4294967295)
    throw new Error("Expected an unsigned 32-bit seed");
  if (!/^#[0-9a-f]{6}$/i.test(color))
    throw new Error("Expected an RGB hex color");
  if (!["type-a", "type-b", "type-c", "type-d"].includes(family))
    throw new Error("Unknown background family");
  const roles = [
    "gradient",
    "grid",
    ...(family === "type-a" || family === "type-d" ? ["country"] : []),
    "lines",
    ...(family === "type-a" || family === "type-c" ? ["header"] : []),
  ];
  const layers = roles.map((role) => {
    const rng = random(seed, role);
    const component: ComponentId =
      role === "country"
        ? "shapes/country_contour"
        : rng.pick(
            groups[role as keyof typeof groups] as readonly ComponentId[],
          );
    const layerColor =
      role === "gradient"
        ? color
        : related(
            color,
            rng,
            role === "header" || role === "country"
              ? rng.range(0.7, 0.9)
              : rng.range(0.2, 0.4),
          );
    return {
      role,
      component,
      parameters: registry[component].parameters(rng, layerColor),
    } as Layer;
  });
  return { ...options, version, layers };
}
