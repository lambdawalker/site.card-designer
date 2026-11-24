# Card Designer

A standalone card-design website extracted from
[ds.plain_id_card](https://github.com/lambdawalker/ds.plain_id_card).
Edit HTML/Jinja-style templates and CSS, preview procedural backgrounds, select
synthetic portraits, save private design bundles, and download sample cards.
Cognito handles user registration/sign-in. Revocable read-only API tokens let a
local renderer consume saved templates.

## Application

The Svelte studio retains the original layout editor, template bundles, color
controls, seeded backgrounds and sample downloads. Rendering happens in the
browser. The Go API stores designs under the authenticated user's identity and
caches whole Hugging Face Parquet packages in S3. The initial portrait source is
[`lambdaWalker/ds.photo-id`](https://huggingface.co/datasets/lambdaWalker/ds.photo-id).
Missing identity text uses placeholders; machine-readable sample assets are
created in JavaScript from synthetic payloads.

This extraction includes the designer and sample downloads. Large export sessions,
YOLO datasets and the original Python generation pipeline remain in the source
project. See [extraction provenance](docs/extraction-provenance.md).

## AWS deployment

Run the root launcher from a checkout:

```bash
./deploy.sh
```

On Windows:

```bat
deploy.bat
```

Both launchers run the same Go deployment program. The wizard collects the domain
at deployment time and keeps settings scoped to the selected project/environment.
Build and preview precede apply. AWS credentials belong to the operator's SSO
profile locally and an assumed OIDC role in CI. Never commit credentials, the
local vault or Pulumi state.

Infrastructure consists of CloudFront, a private website S3 bucket, API Gateway,
a Go Lambda, a private design/package-cache bucket, a DynamoDB token table and a
Cognito user pool. Pulumi owns the stack; its state is stored in S3 separately
from application data. See [deployment instructions](deploy/README.md) for
prerequisites, domain/TLS setup, saved settings, CI and recovery.

## Renderer integration

Sign in, save a bundle and create an API token. Supply the token as a bearer header
to retrieve your own bundles. Personal tokens cannot edit templates or create
other tokens. Keep the token in an environment variable on the renderer machine.

See [API documentation](docs/api.md) and the [renderer examples](examples/).
Download each portrait package once and reuse its rows locally. Pin the dataset
revision and preserve the bundle source when reproducible samples matter.

## Development and verification

Each Go component has an independent module. Run its tests from its directory:

```bash
(cd backend && go test ./...)
(cd deploy && go test ./...)
(cd infra && go test ./...)
(cd packages/drawing-tools && npm ci && npm run check && npm test)
(cd web && npm ci && npm run build && npm test)
```

The browser renderer uses the extracted drawing package under `packages/`.
The build copies `assets/catalog.json` into the backend's embedded catalog so
browser and API template sources agree. Cloud provisioning is intentionally
separate from local tests; a successful build does not establish that an AWS
stack has been deployed.
