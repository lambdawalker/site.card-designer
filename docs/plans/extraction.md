# Standalone Card Designer

Confirmed scope: extract designer from ds.plain_id_card, preserve editable templates and procedural backgrounds, Cognito sign-in and private user designs, browser sample downloads, read-only revocable API tokens for local renderers, HF Parquet package caching, Go backend and Pulumi AWS deployment with domain prompt. Original repository remains untouched.

Architecture: Svelte frontend retains designer. Client-side Jinja-compatible rendering runs in an isolated worker; sandboxed preview HTML. Go Lambda behind API Gateway validates Cognito access tokens and hashed personal tokens. S3 holds per-user bundle JSON and shared immutable HF package cache; DynamoDB stores hashed token records. Frontend served from private S3 via CloudFront. Pulumi provisions infrastructure. Domain and DNS supplied at deploy time; generated URL supported. No production apply during implementation.

Contracts:
- GET /api/config public: {cognito_domain,client_id,redirect_uri,logout_uri,region,user_pool_id}.
- Authorization: Bearer Cognito access JWT or cd_<random>. Tokens read-only except users may manage tokens using Cognito login.
- GET/POST /api/bundles, GET/PUT/DELETE /api/bundles/{id}, POST /api/bundles/{id}/clone. Bundle {id,name,layouts:[design...]}; list [{id,name,layout_count}]. Layout opaque validated JSON, bounded sizes, stable layout_id.
- GET /api/tokens -> metadata array; POST {name} -> {id,token,name,expires_at}; DELETE /api/tokens/{id}. Expiry 90 days; owner-only revoke; token plaintext returned once, sha256 stored.
- GET /api/datasets -> [{id:"photo-id",dataset:"lambdaWalker/ds.photo-id",name:"Synthetic portraits",count:0,reference:"lambdaWalker/ds.photo-id"}]. Fixed allowlisted dataset for initial release.
- GET /api/datasets/photo-id/packages -> {revision,packages:[{path,size}]} using immutable full commit SHA; cached discovery 5 minutes.
- GET /api/datasets/photo-id/package?path=...&revision=... -> {revision,path,rows:[{id,prompt,parameters,image:{bytes:<base64>,path},image_transparent:{bytes:<base64>,path}}]}. Whole package fetched from HF at immutable revision, cached in S3; bounded request/response/download size. Validate path and revision; no arbitrary remote URLs. Token can read packages.
- Frontend adapts legacy local dataset UI to package pagination, maps rows locally, uses embedded data images. Built-in catalog shipped as JSON from source templates; read-only API /api/templates and /api/templates/{id} serves same catalog.
- GET /api/v1/bundles and /api/v1/bundles/{id} aliases for local renderer, bundle source plus rendering contract documented. Runtime and example client provided in repo.

Tasks:
1. Go API/auth/store/HF package cache plus security and caching tests (backend/).
2. Extract frontend and drawing package, generate catalog, add auth/token UI, local rendering and placeholders/codes, remove batch exporter; preserve sample download (web/,packages/,assets/).
3. Pulumi infrastructure and Go deploy wizard following agents.prompts; launchers, workflows, environment isolation and deploy docs (infra/,deploy/,deploy.sh,deploy.bat,.github/).
4. Integration verification, security review, README/API/local-renderer examples, push task branch and open PR.

Verification: Go unit/race tests (ownership, token rejection/revocation, download caching and HF path validation), frontend build and node tests, browser smoke all built-in templates and PNG downloads, Pulumi compilation/mocks, deployment wizard tests. AWS live deployment requires operator credentials and deploy-time configuration; report honestly if not run.
