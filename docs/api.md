# Renderer API

The site exposes the same-origin `/api` API. Use HTTPS for deployed sites. Sign in
through Cognito in the website, open **API tokens**, and create a token. Copy the
plaintext token immediately; it is returned only once. Tokens expire after 90 days
and can be revoked from the same panel. Keep them in environment variables or your
local secret store, never in a template, URL, committed configuration or browser
preview context.

Send `Authorization: Bearer <token>` on each request. Personal API tokens are
read-only and resolve to the user who created them. Cognito access tokens authorize
interactive edits. Neither caller can access another user's saved bundles.

```bash
# Set CARD_DESIGNER_URL and CARD_DESIGNER_TOKEN in your environment first.
curl --fail --silent --show-error \
  -H "Authorization: Bearer ${CARD_DESIGNER_TOKEN}" \
  "${CARD_DESIGNER_URL}/api/v1/bundles"

curl --fail --silent --show-error \
  -H "Authorization: Bearer ${CARD_DESIGNER_TOKEN}" \
  "${CARD_DESIGNER_URL}/api/v1/bundles/BUNDLE_ID"
```

## Designs and templates

| Method and path | Result |
| --- | --- |
| `GET /api/v1/bundles` | Your saved bundle summaries |
| `GET /api/v1/bundles/{id}` | Full saved bundle, including each layout's HTML template, CSS and rendering settings |
| `GET /api/templates` | Built-in template catalog |
| `GET /api/templates/{id}` | One built-in template |

The interactive editor also uses `/api/bundles` as an unversioned alias. Write
methods and `/api/tokens` management require a Cognito login, not a personal token.
Do not interpret a list position as a permanent bundle or layout identifier.

Layouts retain their source, dimensions, seed, theme/background settings and
selected sample. The local renderer must evaluate the same template context and
procedural drawing functions as the website. Use the included renderer example
instead of treating raw HTML/Jinja source as a ready-to-display HTML document.
Store the fetched bundle alongside outputs when reproducibility matters; later
edits to a saved bundle change subsequent API responses.

## Dataset packages

`GET /api/datasets/photo-id/packages` returns an immutable revision and a list of
Parquet package paths. Pass that exact revision and path to:

```
GET /api/datasets/photo-id/package?revision=<full-commit>&path=<encoded-path>
```

The API downloads a complete package on a cache miss, retains it in S3, and returns
its decoded rows. Image bytes in JSON are base64 encoded; `parameters` is the
source dataset's JSON text. All rows in a downloaded package can be reused locally.
The shared package cache is separate from private user designs. Only the configured
portrait dataset is accepted; this endpoint is not a general URL-fetch proxy.

Retain `(dataset, revision, package path, record id)` when saving local render input.
New uploads can alter listing order. Use record IDs instead of ordinal row indexes.
Refresh discovery when you want newly uploaded packages, rather than for each card.
Respect HTTP errors and retry transient failures with backoff; do not retry invalid
credentials or missing templates indefinitely.

## Sample data

Portrait images and prompts come from the synthetic Hugging Face dataset.
Missing identity values use editable placeholders. QR, Code 128 and PDF417 assets
are generated from seeded synthetic payloads in JavaScript. They are visual sample
assets and do not represent verified identity data.

## Browsing and bounds

The studio searches metadata in packages already loaded in the current browsing
session. It does not scan every remote package for each keystroke. Opening more
packages expands the searchable set. Saved selections include their immutable
package location so reopening a design can retrieve the same portrait.

Bundles are limited to 100 layouts and 4 MiB serialized JSON. Portrait package
responses are limited to 5 MiB and 500 rows; oversized packages return an error.
These bounds keep responses within the synchronous Lambda/API request path.
