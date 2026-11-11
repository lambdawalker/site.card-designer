import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import nunjucks from "nunjucks";
import { renderContext, seeded } from "../client-renderer.js";
import { validTransaction } from "../auth.js";
const catalog = JSON.parse(
  readFileSync(new URL("../../assets/catalog.json", import.meta.url)),
);
test("all shipped template variants render with explicit placeholders and finite CSS", () => {
  assert.equal(catalog.length, 16);
  for (const design of catalog) {
    const context = renderContext(design);
    context.env = { theme: { text_color: { to_rgb_hex: () => "#000000" } } };
    const html = nunjucks.renderString(design.template, context),
      css = nunjucks.renderString(design.css, context);
    assert.ok(html.includes("—"), design.id);
    assert.ok(!/NaN|:\s*px|undefined/.test(css), design.id);
    assert.ok(!html.includes("{{"), design.id);
  }
});
test("seed and synthetic machine codes replay deterministically", () => {
  const design = catalog[0];
  assert.deepEqual(renderContext(design), renderContext(design));
  assert.notDeepEqual(
    renderContext({ ...design, seed: 43 }).assets,
    renderContext(design).assets,
  );
  for (const field of ["qrcode", "barcode", "pdf417"]) {
    const url = renderContext(design).assets[field];
    assert.match(Buffer.from(url.split(",")[1], "base64").toString(), /<svg/);
  }
  assert.equal(seeded(10)(), seeded(10)());
});
test("missing identity values remain placeholders; supplied identity values stay unchanged", () => {
  const c = renderContext(
    { ...catalog[0], common: { issuer: "Demo" } },
    { full_name: "A < B" },
  );
  assert.equal(c.identity.full_name, "A < B");
  assert.equal(c.identity.birth_date, "—");
  assert.equal(c.common.issuer, "Demo");
  assert.ok(c.missing.includes("birth_date"));
});
test("OAuth state rejects mismatch, missing, future and expired transactions", () => {
  const tx = { state: "a".repeat(32), verifier: "secret", created: 1000 };
  assert.equal(validTransaction(tx, tx.state, 1001), true);
  assert.equal(validTransaction(tx, "b".repeat(32), 1001), false);
  assert.equal(validTransaction(null, tx.state, 1001), false);
  assert.equal(validTransaction(tx, tx.state, 999), false);
  assert.equal(validTransaction(tx, tx.state, 601000), false);
});
