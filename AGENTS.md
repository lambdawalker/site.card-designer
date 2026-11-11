# Card Designer contributor guide

This is a standalone website, not a reusable Go library. Read README.md and
docs/plans/extraction.md before changing subsystem contracts.

- `web/`: Svelte designer, Cognito PKCE login and browser rendering.
- `packages/`: extracted procedural drawing tools.
- `assets/`: built-in template catalog used by the web client and API.
- `backend/`: Go API, Cognito/PAT authentication, private designs and HF cache.
- `infra/`: Pulumi-owned AWS infrastructure.
- `deploy/`: shared Go deployment program used by both root launchers.

Preserve ownership boundaries: user identifiers come from verified authentication,
never request bodies. Personal tokens are read-only, hashed at rest, expiring and
revocable. Never persist or log plaintext bearer tokens. Preview frames must stay
sandboxed; user-authored templates must not execute in the application window.

Cache HF packages by repository, immutable revision and full path. Do not fetch
arbitrary user-supplied URLs or treat row ordinals as stable record identifiers.
Use synthetic QR/barcode payloads and explicit placeholders for absent fields.

Do not restore batch export sessions to this project. Local renderers consume the
versioned read API and shared browser rendering implementation.

Keep build, preview and apply distinct; never apply AWS changes merely to test.
Test locally with isolated stores/fake cloud services, run relevant module suites,
and report cloud validation separately. Default delivery is a task branch and PR;
merging or publishing requires explicit user instruction.
