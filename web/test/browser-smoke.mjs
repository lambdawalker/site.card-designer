import { chromium } from "playwright";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const origin = process.env.CARD_SERVER || "http://127.0.0.1:5173";
const browser = await chromium.launch({
  executablePath: process.env.CARD_CHROMIUM || undefined,
  args: ["--no-sandbox"],
});
try {
  const page = await browser.newPage();
  await page.goto(origin + "/renderer.html");
  await page.waitForFunction(() => window.rendererReady);
  for (const design of JSON.parse(
    readFileSync(new URL("../../assets/catalog.json", import.meta.url)),
  )) {
    const rendered = await page.evaluate(
      (design) => window.renderSpecimen(design),
      design,
    );
    assert.deepEqual(
      rendered.png.slice(0, 8),
      [137, 80, 78, 71, 13, 10, 26, 10],
      design.id,
    );
    const pixels = await page.evaluate(async (png) => {
      const image = new Image();
      image.src = URL.createObjectURL(
        new Blob([new Uint8Array(png)], { type: "image/png" }),
      );
      await image.decode();
      URL.revokeObjectURL(image.src);
      const canvas = document.createElement("canvas");
      canvas.width = image.width;
      canvas.height = image.height;
      const ctx = canvas.getContext("2d");
      ctx.drawImage(image, 0, 0);
      const data = ctx.getImageData(0, 0, image.width, image.height).data;
      let opaque = 0;
      const colors = new Set();
      for (let i = 0; i < data.length; i += 4) {
        if (data[i + 3]) opaque++;
        colors.add(Array.from(data.slice(i, i + 4)).join(","));
      }
      return {
        width: image.width,
        height: image.height,
        opaque,
        colors: colors.size,
      };
    }, rendered.png);
    console.log(design.id, rendered.png.length, pixels);
    assert.ok(
      pixels.opaque > (pixels.width * pixels.height) / 2,
      design.id + " is mostly transparent",
    );
    assert.ok(pixels.colors > 20, design.id + " has no rendered content");
    assert.ok(rendered.png.length > 5000, design.id);
    // Embedded PNG/SVG base64 may legitimately contain the letters "NaN".
    // Validate rendered markup/CSS, not the opaque image payloads.
    const markup = rendered.html.replace(/data:[^"'()\s<>]+/g, "[embedded]");
    assert.ok(!markup.includes("NaN"), design.id);
  }
  await page.goto(origin);
  await page.getByRole("button", { name: "Sign in", exact: true }).waitFor();
  await page.waitForFunction(() =>
    document
      .querySelector('iframe[title="Card preview"]')
      ?.srcdoc.includes("view-port"),
  );
  const download = page.waitForEvent("download");
  await page
    .getByRole("button", { name: "Download card PNG", exact: true })
    .click();
  assert.match((await download).suggestedFilename(), /\.png$/);
  assert.equal(
    await page.locator('iframe[title="Card preview"]').getAttribute("sandbox"),
    "",
  );
  console.log(
    "All catalog variants rendered PNG; guest shell and sample download passed.",
  );
} finally {
  await browser.close();
}
