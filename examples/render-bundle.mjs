// npm install playwright; CARD_DESIGNER_TOKEN=cd_... node examples/render-bundle.mjs https://your-site BUNDLE_ID output
import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
const [site, bundleId, directory = "output"] = process.argv.slice(2),
  token = process.env.CARD_DESIGNER_TOKEN;
if (!site || !bundleId || !token)
  throw Error(
    "Usage: CARD_DESIGNER_TOKEN=cd_... node render-bundle.mjs HTTPS_SITE BUNDLE_ID [OUTPUT]",
  );
const origin = new URL(site);
if (
  origin.protocol !== "https:" &&
  !(
    origin.protocol === "http:" &&
    ["localhost", "127.0.0.1", "[::1]"].includes(origin.hostname)
  )
)
  throw Error("Use HTTPS (HTTP allowed only for localhost)");
const packageCache = new Map();
const response = await fetch(
  new URL("/api/v1/bundles/" + encodeURIComponent(bundleId), origin),
  { redirect: "error", headers: { Authorization: "Bearer " + token } },
);
if (!response.ok) throw Error("Bundle request failed: " + response.status);
const bundle = await response.json();
const browser = await chromium.launch(
  process.env.CHROMIUM_PATH
    ? { executablePath: process.env.CHROMIUM_PATH }
    : {},
);
try {
  const page = await browser.newPage();
  await page.goto(new URL("/renderer.html", origin).href);
  await page.waitForFunction(() => window.rendererReady);
  await mkdir(directory, { recursive: true });
  for (const [index, design] of bundle.layouts.entries()) {
    let row;
    const source = design.sample_source;
    if (source) {
      const key = source.revision + ":" + source.path;
      if (!packageCache.has(key)) {
        const packageResponse = await fetch(
          new URL(
            "/api/datasets/photo-id/package?" +
              new URLSearchParams({
                path: source.path,
                revision: source.revision,
              }),
            origin,
          ),
          { redirect: "error", headers: { Authorization: "Bearer " + token } },
        );
        if (!packageResponse.ok)
          throw Error(
            "Portrait package request failed: " + packageResponse.status,
          );
        packageCache.set(key, await packageResponse.json());
      }
      const data = packageCache.get(key);
      row =
        data.rows.find((row) => String(row.id) === String(source.id)) ||
        data.rows[source.index];
    }
    const result = await page.evaluate(
      ({ design, row }) => window.renderSpecimen(design, row),
      { design, row },
    );
    await writeFile(
      directory + "/" + String(index + 1).padStart(2, "0") + ".png",
      Buffer.from(result.png),
    );
  }
} finally {
  await browser.close();
}
