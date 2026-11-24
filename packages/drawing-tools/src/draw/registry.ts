import * as gradient_linear_gradient from "./gradient/linear_gradient/index.ts";
import * as gradient_radial_gradient from "./gradient/radial_gradient/index.ts";
import * as gradient_sin from "./gradient/sin/index.ts";
import * as gradient_voronoid from "./gradient/voronoid/index.ts";
import * as grid_dots_grid from "./grid/dots_grid/index.ts";
import * as grid_hexagon_grid from "./grid/hexagon_grid/index.ts";
import * as grid_triangle_grid from "./grid/triangle_grid/index.ts";
import * as grid_shapes_grid from "./grid/shapes_grid/index.ts";
import * as grid_concentric_polygons from "./grid/concentric_polygons/index.ts";
import * as lines_parallel_lines from "./lines/parallel_lines/index.ts";
import * as lines_parallel_sin from "./lines/parallel_sin/index.ts";
import * as lines_parallel_sawtooth from "./lines/parallel_sawtooth/index.ts";
import * as lines_square_wave from "./lines/square_wave/index.ts";
import * as lines_wavy from "./lines/wavy/index.ts";
import * as lines_dotted_waves from "./lines/dotted_waves/index.ts";
import * as header_square_header from "./header/square_header/index.ts";
import * as header_curved_header from "./header/curved_header/index.ts";
import * as header_sin_header from "./header/sin_header/index.ts";
import * as country from "./shapes/country_contour.ts";
export const registry = {
  "gradient/linear_gradient": gradient_linear_gradient,
  "gradient/radial_gradient": gradient_radial_gradient,
  "gradient/sin": gradient_sin,
  "gradient/voronoid": gradient_voronoid,
  "grid/dots_grid": grid_dots_grid,
  "grid/hexagon_grid": grid_hexagon_grid,
  "grid/triangle_grid": grid_triangle_grid,
  "grid/shapes_grid": grid_shapes_grid,
  "grid/concentric_polygons": grid_concentric_polygons,
  "lines/parallel_lines": lines_parallel_lines,
  "lines/parallel_sin": lines_parallel_sin,
  "lines/parallel_sawtooth": lines_parallel_sawtooth,
  "lines/square_wave": lines_square_wave,
  "lines/wavy": lines_wavy,
  "lines/dotted_waves": lines_dotted_waves,
  "header/square_header": header_square_header,
  "header/curved_header": header_curved_header,
  "header/sin_header": header_sin_header,
  "shapes/country_contour": country,
};

export type ComponentId = keyof typeof registry;
export const groups = {
  gradient: [
    "gradient/linear_gradient",
    "gradient/radial_gradient",
    "gradient/sin",
    "gradient/voronoid",
  ],
  grid: [
    "grid/dots_grid",
    "grid/hexagon_grid",
    "grid/triangle_grid",
    "grid/shapes_grid",
    "grid/concentric_polygons",
  ],
  lines: [
    "lines/parallel_lines",
    "lines/parallel_sin",
    "lines/parallel_sawtooth",
    "lines/square_wave",
    "lines/wavy",
    "lines/dotted_waves",
  ],
  header: ["header/square_header", "header/curved_header", "header/sin_header"],
} as const;
export type Layer = {
  [K in ComponentId]: {
    role: string;
    component: K;
    parameters: ReturnType<(typeof registry)[K]["parameters"]>;
  };
}[ComponentId];
export function paintLayer(ctx: CanvasRenderingContext2D, layer: Layer) {
  // The discriminated recipe is validated at the public replay boundary.
  const painter = registry[layer.component].paint as (
    ctx: CanvasRenderingContext2D,
    p: Layer["parameters"],
  ) => void | Promise<void>;
  return painter(ctx, layer.parameters);
}
