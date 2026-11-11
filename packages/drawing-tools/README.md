# Browser drawing tools

A TypeScript and Canvas/WebGL adaptation of the Python drawing library. The package
is local to this repository. The IDE and bulk exporter import the same implementation.

## Structure

Modules mirror Python under src/draw. Every gradient, grid, line pattern and header
has parameters.ts, paint.ts, create.ts and index.ts. Parameters are serializable.
Painters draw onto a supplied CanvasRenderingContext2D. Creators allocate a canvas.
Country artwork and contour compositing live under assets and draw/shapes.

The registry contains four gradients, five grids, six line patterns, three headers,
and one country contour painter. Background composition lives under draw/card_background.

| Family | Ordered layers |
| --- | --- |
| type-b | Gradient, grid, lines |
| type-c | Gradient, grid, lines, header |
| type-d | Gradient, grid, country, lines |
| type-a | Gradient, grid, country, lines, header |

## Use

~~~typescript
import {create, replay} from '@lambdawalker/drawing-tools';

const {canvas, recipe} = await create({
  family: 'type-a', seed: 42, color: '#95b5c7', width: 640, height: 400,
});
const repeated = await replay(JSON.parse(JSON.stringify(recipe)));
// Optional backing dimensions: the recipe and all layout coordinates stay 640 × 400.
const sharp = await replay(recipe, {width: 1280, height: 800});
~~~

Each layer has its own random stream derived from its role. Adding a country or header
keeps the selected gradient, grid and lines consistent for a given seed. Recipes store
the package version, all resolved parameters, color, seed, dimensions, component IDs
and country filename. Replay validates parameters before painting.

The current recipe version is 0.1.0. Future algorithm changes must retain an old
renderer or introduce a new version with explicit migration. Pin browser, fonts,
assets and graphics configuration when comparing pixels across machines.

## Rendering differences from Python

These are browser adaptations with their own deterministic random sequence and
parameter ranges. Python seeds and pixels are not interchangeable with browser seeds.

Colors use HSLuv to derive related shades. Canvas uses browser antialiasing and alpha
compositing. The sinusoidal gradient computes pixels directly. Voronoi uses the
upstream shader adapted to WebGL and requires WebGL availability.

The wavy and dotted-wave painters use seeded gradient noise in place of Python's
pnoise2 implementation and local unseeded random operations. Both follow noise-driven
paths, with variable-width segments for dotted waves. Grid placement and sampling
follow browser geometry. Country outlines use alpha-mask dilation and Canvas blur
in place of OpenCV contour extraction and Gaussian blur. These differences require
visual review before mixing generated browser images with existing Python datasets.

Country assets load on demand and are cached. The IDE caches the most recent finished
or pending background so HTML edits reuse it. WebGL resources are released after each
Voronoi paint.

## Verification

Run npm ci, npm run check and npm test in this package. Install Chromium using
npx playwright install chromium, then run npm run test:browser after installing the
IDE's web dependencies. CARD_CHROMIUM can select an existing executable.

The browser test starts Vite, paints every component, checks all families at landscape
and portrait dimensions, compares serialized recipe replay, changes seeds, and rejects
invalid spacing. It uses software WebGL flags for CI. The ordinary unit test checks
recipe serialization and independence between layers.

Country assets and the shader retain upstream attribution in THIRD_PARTY.md.

`fitHeader(recipe, bounds, padding = 12)` returns a cloned recipe whose header layers cover a measured `{left, top, right, bottom}` region in logical card pixels. It compensates for curved/wavy edges and records `header_layout` metadata. The IDE calls it after rendering text and CSS, before painting the background; content is remeasured on each render. Other layer parameters remain unchanged. Passing `null` leaves layer geometry unchanged and clears old measurement metadata. Invalid/out-of-card measurements are rejected.
