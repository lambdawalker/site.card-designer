import { create as compose } from "./create.ts";
import type { Options } from "./parameters.ts";
export function create(options: Omit<Options, "family">) {
  return compose({ ...options, family: "type-c" });
}
