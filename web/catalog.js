import catalog from "../assets/catalog.json" with { type: "json" };
export { catalog };
export const stockBundles = Object.values(
  Object.groupBy(catalog, (item) => item.bundle_id),
).map((layouts) => ({
  id: "stock:" + layouts[0].bundle_id,
  name: layouts[0].bundle_name,
  layout_count: layouts.length,
  layouts,
}));
