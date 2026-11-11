import { surface, context } from "../surface.ts";
import type { Size } from "../surface.ts";
import { paintLayer, registry } from "../registry.ts";
import { random } from "../../random/seeded.ts";
import { parameters, version } from "./parameters.ts";
import type { Options, Recipe } from "./parameters.ts";

function validate(recipe: Recipe) {
  parameters(recipe);
  if (recipe.version !== version)
    throw new Error("Unsupported background recipe version " + recipe.version);
  if (!Array.isArray(recipe.layers) || recipe.layers.length > 5)
    throw new Error("Invalid recipe layers");
  for (const layer of recipe.layers) {
    if (!Object.hasOwn(registry, layer.component))
      throw new Error("Unknown drawing component");
    if (!layer.parameters || typeof layer.parameters !== "object")
      throw new Error("Invalid layer parameters");
    const schema = registry[layer.component].parameters(
      random(0),
      recipe.color,
    );
    const values = layer.parameters as unknown as Record<string, unknown>;
    for (const [name, example] of Object.entries(schema)) {
      const value = values[name];
      if (typeof value !== typeof example)
        throw new Error("Invalid parameter " + name);
      if (typeof value === "string") {
        if (
          name === "shape" &&
          !["circle", "square", "triangle", "polygon", "star"].includes(value)
        )
          throw new Error("Invalid grid shape");
        if (
          (name === "color" ||
            name.endsWith("_color") ||
            name === "color_a" ||
            name === "color_b") &&
          !CSS.supports("color", value)
        )
          throw new Error("Invalid layer color");
      }
      if (typeof value === "number") {
        if (!Number.isFinite(value) || Math.abs(value) > 4294967295)
          throw new Error("Invalid numeric parameter " + name);
        if (
          [
            "spacing",
            "wavelength",
            "radius",
            "thickness",
            "frequency",
            "step",
          ].includes(name) &&
          (value < 0.5 || value > 4096)
        )
          throw new Error(name + " must be between 0.5 and 4096");
        if (name === "scale" && (value < 0.0001 || value > 100))
          throw new Error("Invalid noise scale");
        if (
          name === "num_lines" &&
          (!Number.isInteger(value) || value < 1 || value > 300)
        )
          throw new Error("Line count must be between 1 and 300");
        if (name === "mod" && (value < 1 || value > 4096))
          throw new Error("Invalid noise modulation");
        if (
          ["sides"].includes(name) &&
          (!Number.isInteger(value) || value < 3 || value > 32)
        )
          throw new Error("Invalid polygon sides");
        if (["height"].includes(name) && (value < 0 || value > 1))
          throw new Error("Invalid header height");
      }
    }
  }
}
export async function replay(recipe: Recipe, target?: Size) {
  validate(recipe);
  const canvas = surface(recipe.width, recipe.height, target),
    ctx = context(canvas);
  for (const layer of recipe.layers) {
    ctx.save();
    try {
      await paintLayer(ctx, layer);
    } finally {
      ctx.restore();
    }
  }
  return { canvas, recipe };
}
export async function create(options: Options, target?: Size) {
  return replay(parameters(options), target);
}
