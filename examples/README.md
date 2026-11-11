# Local rendering

Sign in to Studio and create a named read-only API token. Tokens are shown once and expire in 90 days. Install Playwright in your local Node environment (`npm install --no-save playwright` and `npx playwright install chromium`).

```
CARD_DESIGNER_TOKEN=cd_... node examples/render-bundle.mjs https://your-site.example BUNDLE_ID output
```

Use `CHROMIUM_PATH` to choose an already-installed browser. The example fetches `/api/v1/bundles/{id}` from Node using the token, then supplies only design JSON to the site's public `/renderer.html`. No credentials enter the browser template renderer. Each layout produces one PNG. Saved procedural recipes and seeds replay. Missing dataset identity fields use placeholders; QR, Code128 and PDF417 encode synthetic `SYNTHETIC-SPECIMEN-{seed}` data. Do not use specimens as identity credentials.

The browser renderer exposes `window.rendererReady` and `window.renderSpecimen(design)`, returning `{html,png}` where `png` is a byte array. Rendering uses a fresh bounded worker per layout. Template source plus CSS is capped at 500 KB, rendering times out after three seconds, and previews have a restrictive CSP and sandbox. Nunjucks supports Jinja expressions used by the shipped catalog; arbitrary Python methods are not supported.
