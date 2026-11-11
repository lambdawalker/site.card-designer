# Card Designer API

Go 1.24+ module `github.com/lambdawalker/site.card-designer/backend`. Standard HTTP server and API Gateway v2 Lambda use the same handler. There is no authentication bypass, including local mode.

```
go test -race ./...
go run ./cmd/server
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/lambda
```

Required runtime environment: `AWS_REGION`, `DATA_BUCKET`, `TOKEN_TABLE`, `COGNITO_USER_POOL_ID`, `COGNITO_CLIENT_ID`. Public configuration also reads `COGNITO_DOMAIN`, `REDIRECT_URI`, `LOGOUT_URI`. `HF_TOKEN` is optional and stays server-side; AWS credentials use the SDK default provider chain (local SSO/profile or Lambda role). `LISTEN_ADDR` defaults to `127.0.0.1:8080` in the HTTP server only. Copy `../assets/catalog.json` to `internal/catalog/catalog.json` before building whenever the built-in catalog changes.

Only GET `/api/config`, `/health` and `/api/health` are public. All other routes require `Authorization: Bearer <Cognito access JWT or cd_...>`. JWT verification checks RS256 signature, issuer, expiry, token_use=access and client_id; JWKS cache expires hourly and unknown-key refreshes are limited to once per minute. Each personal-token request uses a consistent DynamoDB read, so explicit revocation takes effect immediately. Token plaintext is returned once at creation; only SHA-256 is stored, with a 90-day expiry enforced independently of DynamoDB TTL.

Personal tokens may only GET bundles, templates and dataset packages. Token management, all writes and clones require Cognito login. Every user S3 key is scoped to `users/<sha256-Cognito-sub>/bundles/<id>.json`; user-controlled IDs cannot insert paths. API `/api/v1/bundles` and `/api/v1/bundles/{id}` alias the bundle read interface for local rendering. `X-Card-Designer-API-Version: 1` identifies the rendering contract.

DynamoDB needs string partition key `token_hash`, numeric TTL field `expires_at`, and an ALL-projection GSI `owner-index` with string partition key `owner`. IAM needs GetItem/PutItem/DeleteItem on the table and Query on its owner index. S3 needs GetObject/PutObject/DeleteObject under users/, GetObject/PutObject under cache/hf/, and prefix-constrained ListBucket for users/. Stored objects request AES256 encryption; the bucket and transport policies are deployment responsibilities.

Bundles are JSON `{id,name,layouts:[opaque design objects]}`. Names are 1–200 bytes, IDs use letters/digits/underscore/hyphen, and layouts number 1–100. Missing layout_id is generated; duplicate IDs reject the save. Raw JSON design fields, including background artwork and sample_source, are preserved. The serialized bundle cap is 4 MiB. PUT requires an existing owner-scoped bundle. Clone requires an existing source; an optional posted bundle uses its current unsaved draft. GET collection returns `{id,name,layout_count}` summaries. There are no conditional edit headers; last completed write wins.

HF access is fixed to `lambdaWalker/ds.photo-id`. Discovery resolves main to a full immutable 40-character SHA and caches the listing for five minutes. Tree pagination is bounded to 10 pages / 10,000 entries and reports overflow. Only `data/train/<safe-filename>.parquet` is allowed. Whole-package JSON is cached persistently under `cache/hf/photo-id/<SHA>/...`; historical pinned revisions remain retrievable. Cache errors fail closed instead of repeatedly downloading. One package operation runs per process at a time, with context cancellation while waiting. Download cap: 24 MiB; row cap: 500; response cap: 5 MiB; discovery page cap: 2 MiB. Oversized packages are excluded from discovery or rejected. Responses retain only id, prompt, parameters, image and image_transparent; image bytes serialize as base64. HTTP transient failures and 429/5xx retry at most three attempts, with bounded Retry-After waits. Requests have a 25-second API context and 20-second upstream HTTP timeout. Redirects strip Authorization.

Unit tests use fake object/token stores and a local JWKS server; they cover tenant isolation, read-only permissions, hashing, expiry, owner revocation, signature and claim rejection, and discovery/S3 caching and path validation. `LIVE_HF=1 go test -v ./internal/api -run TestLiveHF` opts into a read-only public HF smoke and proves actual Parquet decoding and base64 image output. AWS integration and deployed Cognito/Lambda acceptance require operator credentials and are not claimed by these tests.
