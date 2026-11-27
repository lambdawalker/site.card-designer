# Composer deployment migration

Scope: align the standalone Card Designer deployment with Infra Composer. Keep the
application API, resource logical names, Pulumi project `card-designer`, stack
`<project>-<environment>`, existing state URL and secrets provider unchanged.
Actual AWS deployment status is unknown; no live mutation is part of this work.

Plan:
1. Add narrowly scoped shared CLI support for nested Go modules, prepared web
   artifacts and project-derived stack names; verify it in Composer.
2. Replace the private wizard with a pinned Composer CLI, declarative manifest
   and a local asset builder. Adapt Pulumi config and CI to the shared contract.
3. Preserve legacy settings/state with explicit migration guidance and guards;
   run local builds, tests and workflow checks; deliver linked PRs.

At the start of the migration, the shared CLI lacked a nested build directory and
prepared-asset hook. The extension now runs a declared repository Go package
without a shell, caches its declared input/output contents, and binds those
outputs to preview approval.
No application authentication or infrastructure topology redesign is planned.

Requirement/evidence matrix is completed with the implementation below.

## Implementation and evidence

The shared extension is [Composer PR #50](https://github.com/lambdawalker/go.infra.composer/pull/50).
The initial consumer migration pinned commit
`32d1abf8d9b0220b72fcf4bc388999156f427fcf` through a checksummed Go pseudo-version.
Composer PR #50 and Card Designer PR #2 were subsequently merged, followed by
browser vault setup in Composer #51 and Card Designer #3. See the upgrade record
below for the current dependency; commit pins are not claimed stable releases.

| Requirement ID and source section | Role/target | Status | Implementation reference | Validation evidence | Remaining work |
| --- | --- | --- | --- | --- | --- |
| D01 [Shared CLI](https://github.com/lambdawalker/agents.prompts/blob/main/prompts/libraries/go-aws-common.md#deployment-wizard-requirements-for-deployable-go--aws-projects) | Application tooling | Implemented | `deploy.json`, `deploy/go.mod`, `deploy/main.go` | Normal root launcher loads the pinned CLI and manifest from another cwd | None |
| D02 Portable entrypoints | Local/CI | Implemented | `deploy.sh`, `deploy.bat`, `.github/workflows/ci.yml` | POSIX help/build from `/tmp`; shared CLI Windows cross-build; native Windows CI configured | Native runner results reported separately |
| D03 Shared terminal/browser flow | Tooling | Implemented | Delegation to Composer `infra-deploy` | Shared CLI/wizard tests pass under race detector; manifest validates through normal launcher | No new browser UI implemented; live cloud interaction not exercised |
| D04 Real artifacts and caching | Application | Implemented | `deploy/cmd/assets`, manifest `PrepareBuild`, `BuildWorkingDir` | Frontend built; ZIP contains executable ELF ARM64 bootstrap; second build reused both artifacts | None |
| D05 Preview/apply binding | Shared tooling | Implemented | Composer `deploymentDigest`, refreshed preview, plan/archive checks | Mocked source mutation rejects apply; output tamper, changed input, forced rebuild and preparation failure tests pass | Live preview/apply not run |
| D06 Stack identity and existing resources | Pulumi | Implemented | `{project}-` stack prefix; `infra/main.go` config compatibility and identity output | Existing logical names retained; Pulumi mock contract suite passes | First real adoption requires deliberate review |
| D07 Legacy nonsecret migration | Application tooling | Implemented | `deploy/migrate.go` | Tests preserve original settings/vault bytes, initialized-stack receipt, reject incomplete/conflicting scopes, support retry | Stop all writers and back up records before operator conversion |
| D08 Legacy secret migration | Local vault | Partial | Original vault retained; new Composer vault uses original stack passphrase | Vault preservation tested; no automatic decrypt/re-encrypt attempted | Operator supplies original Pulumi passphrase; old vault recovery remains manual |
| D09 Scoped records, SSO, state, encryption, locking | Shared tooling | Implemented via shared CLI | Composer project/stack/settings/vault paths | Shared CLI mocked flow and safety suites pass | Real account/S3 permissions not checked |
| D10 Domain/TLS inputs | Application | Implemented | Manifest fields and `infra/settings.go` | Generated/custom hostname cases, region/account ARN restrictions and invalid input rejection tested | Operator must own/validate certificate and DNS |
| D11 GitHub environment reconciliation | Tooling | Implemented via shared CLI | Manifest GitHub config and BENCH variable mapping | Shared CLI GitHub tests pass; workflow YAML parsed; same launcher called | Operator configures protected environments and authorizes setup |
| D12 OIDC provider/role provisioning | AWS setup | Partial | Existing-role mode and deployment instructions | Workflow uses OIDC, minimal permissions and matching environment concurrency | Existing reviewed provider/role required; automatic creation not enabled |
| D13 Artifact reports/purposes | Application | Implemented via shared CLI | `ArtifactDescriptions`, allowlisted outputs and Composer report actions | Shared report suites pass; known resources retain exact logical names | Live evidence unavailable; built files appear by Pulumi asset identity |
| D14 Readiness | Application | Implemented | Manifest `HealthCheck`: `siteUrl` + `/api/config` | Shared bounded probe implementation; build/API tests pass | Actual HTTPS probe and end-to-end AWS acceptance not run |
| D15 Shared admin/index and discovery | Application | Not applicable | Single website/API stack | No separate index resources or clients needing environment discovery | None |
| D16 Approval and delivery | Repositories | Implemented | Task branches and linked PRs | Local test/build evidence below | No merges or live deployment authorized |

Verified locally on 2026-11-25:

- Composer `deploycli`: `go test -race ./...`, `go vet ./...`, Windows/amd64 cross-build.
- Card Designer `backend`, `deploy`, `infra`: `go test -race ./...`; `go vet ./...`.
- Root `deploy.sh --help` and `--action build` from outside the checkout; repeated
  build logs verified reuse of prepared artifacts and Lambda archive.
- Drawing tools: TypeScript check and 7 tests. Website: production build and 17 tests.
- Workflow YAML parse, manifest/Pulumi project consistency, `git diff --check`.

The website build retains its existing large-chunk advisory. No application UI
logic changed. Live provisioned/configured/accepted status remains **unknown**;
AWS preview/apply, GitHub environment mutation and live acceptance were **not run**.

Compatibility follow-up: Card Designer explicitly opts into `BindDeploymentInputs`
and `RefreshBeforePreview`. Existing email/OCR manifests retain their original
defaults. Both real consumer Lambda builds and cache reuse passed with the
candidate CLI; core and Pulumi modules are unchanged. See Composer
`deploycli/docs/consumer-compatibility.md` for inspected revisions and limits.

## Composer workspace upgrade 2026-11-27

`deploy/go.mod` now pins the shared CLI at
`v0.0.0-20261127172311-ff065d4b853c` (Composer main after PR #98).
Its reporting dependency is the upstream-selected
`v0.0.0-20261127172010-c983d95dd87b`. Both are verified by `deploy/go.sum`;
there are no local replacements or floating dependencies.

This brings environment tabs, deployment history, drift checks, resource change
and dependency inspection, the multi-level 3D topology and self-contained HTML
reports into the existing launchers. First-run browser vault password creation
and confirmation remain available. `ArtifactAttachments` supplies six exact-name
presentation hints for older inventories missing dependency information.
`deploy/topology_test.go` checks those hints against the pinned Composer resolver
and exported HTML report; consumer-specific compatibility stays in this repo.

The manifest remains version 1. No launcher or workflow changes are required:
both platforms and Actions already compile the tool from `deploy/go.mod`.
Infrastructure source, resource names, stack identity, vault migration, approval
binding, refreshed previews and readiness checks are unchanged. No changes to
Composer or its other consumers are required by this upgrade.

Local verification:

- All three application Go modules: race tests and vet.
- Pinned Composer CLI, browser wizard and reporting packages: race tests.
- Normal launcher help/build from outside the checkout; repeated build reuses
  verified prepared assets and the Lambda archive.
- Chromium smoke test of the normal browser launcher: workspace loads without
  JavaScript errors and first-run vault creation advances to password confirmation.
  The session stops before saving credentials or contacting AWS.
- Drawing tools: TypeScript check and 7 tests; website: 17 tests and production
  build. Existing Svelte accessibility and bundle-size advisories remain.

Native Windows launcher verification also runs in the PR's existing CI matrix.
AWS preview, apply, drift and live readiness were not run; local validation does
not establish that an existing deployment is healthy or ready to apply.
