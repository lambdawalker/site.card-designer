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
    assert.ok(rendered.png.length > 5000, design.id);
    assert.ok(!rendered.html.includes("NaN"), design.id);
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
