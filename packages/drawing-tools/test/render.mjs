import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { setTimeout } from "node:timers/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
const origin = process.env.DRAWING_TEST_URL || "http://127.0.0.1:8766";
const web = fileURLToPath(new URL("../../../web/", import.meta.url));
const server = process.env.DRAWING_TEST_URL
  ? null
  : spawn(
      process.execPath,
      [
        web + "node_modules/vite/bin/vite.js",
        "--host",
        "127.0.0.1",
        "--port",
        "8766",
      ],
      { cwd: web, stdio: "ignore" },
    );
let browser;
try {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(origin)).ok) break;
    } catch {}
    await setTimeout(100);
  }
  browser = await chromium.launch({
    executablePath: process.env.CARD_CHROMIUM || undefined,
    args: [
      "--no-sandbox",
      "--use-gl=angle",
      "--use-angle=swiftshader",
      "--enable-unsafe-swiftshader",
    ],
  });
  const page = await browser.newPage();
  const root = new URL("../", import.meta.url).pathname;
  await page.goto(origin + "/@fs" + root + "test/browser.html");
  await page.waitForFunction(() => window.drawing);
  const report = await page.evaluate(async () => {
    const { create, replay, registry, random, parameters } = window.drawing;
    const results = [];
    for (const [id, component] of Object.entries(registry)) {
      const canvas = document.createElement("canvas");
      canvas.width = 320;
      canvas.height = 200;
      const ctx = canvas.getContext("2d");
      await component.paint(
        ctx,
        component.parameters(random(42, id), "#567890"),
      );
      const pixels = ctx.getImageData(0, 0, 320, 200).data;
      const painted = pixels.filter((v, i) => i % 4 === 3 && v > 0).length;
      if (!painted) throw Error(id + " painted no pixels");
      results.push({ component: id, painted });
    }
    // Render every component at the final backing resolution without changing its recipe.
    for (const [id, component] of Object.entries(registry)) {
      const recipe = parameters({family:"type-b",width:320,height:200,seed:42,color:"#567890"});
      recipe.layers = [{role:"test",component:id,parameters:component.parameters(random(42,id),"#567890")}];
      const before = JSON.stringify(recipe);
      const native = await replay(recipe);
      const sharp = await replay(recipe,{width:640,height:400});
      if(sharp.canvas.width!==640||sharp.canvas.height!==400||JSON.stringify(recipe)!==before)throw Error(id+" changed recipe or target size");
      const down = document.createElement("canvas");down.width=320;down.height=200;
      const ctx=down.getContext("2d");ctx.drawImage(sharp.canvas,0,0,320,200);
      const a=native.canvas.getContext("2d").getImageData(0,0,320,200).data,b=ctx.getImageData(0,0,320,200).data;
      const error=a.reduce((sum,value,index)=>sum+Math.abs(value-b[index]),0)/a.length;
      if(error>(id==="shapes/country_contour"?1:12))throw Error(id+" changed layout at higher resolution: "+error);
      results.push({component:id,resolution:"2x",meanError:error});
    }
    for (const family of ["type-a", "type-b", "type-c", "type-d"]) {
      for (const [width, height] of [
        [320, 200],
        [640, 400],
        [400, 640],
      ]) {
        const options = { family, width, height, seed: 42, color: "#95b5c7" };
        const start = performance.now();
        const a = await create(options);
        const image = a.canvas.toDataURL();
        const b = await replay(JSON.parse(JSON.stringify(a.recipe)));
        if (image !== b.canvas.toDataURL())
          throw Error("Replay mismatch " + family);
        const c = await create({ ...options, seed: 43 });
        if (image === c.canvas.toDataURL())
          throw Error("Seed did not change " + family);
        results.push({
          family,
          width,
          height,
          milliseconds: Math.round(performance.now() - start),
        });
      }
    }
    const invalid = parameters({
      family: "type-b",
      width: 320,
      height: 200,
      seed: 42,
      color: "#95b5c7",
    });
    invalid.layers[1].parameters.spacing = 0;
    // Use a known component so this also checks malicious loop parameters.
    invalid.layers[1] = {
      role: "grid",
      component: "grid/dots_grid",
      parameters: { color: "#123456", radius: 2, spacing: 0, angle: 0 },
    };
    let rejected = false;
    try {
      await replay(invalid);
    } catch {
      rejected = true;
    }
    if (!rejected) throw Error("Invalid recipe was accepted");
    return results;
  });
  assert.equal(report.filter((r) => r.component && !r.resolution).length, 19);
  console.log(JSON.stringify(report, null, 2));
} finally {
  await browser?.close();
  server?.kill();
}
