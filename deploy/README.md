# Deployment with Infra Composer

The root launchers delegate to the exact `go.infra.composer/deploycli` version in
`deploy/go.mod`. `deploy.json` declares this website's inputs, artifacts, stack
identity, readiness check and resource purposes. Composer owns the terminal and
browser wizards, SSO, encrypted vault, state handling, saved-plan approvals,
GitHub configuration, and JSON/Markdown/HTML artifact reports.

The small `deploy/` adapter handles old flag aliases and offline record migration.
`deploy/cmd/assets` installs the drawing tools and website dependencies, builds
Svelte, and copies the shared template catalog into the backend. It has no AWS
or approval logic. The CLI builds and caches the Linux/ARM64 Lambda archive.

## Run

Install Go 1.25+, Node **24**/npm, AWS CLI v2 and Pulumi CLI v3. Configure an AWS
IAM Identity Center profile with `aws configure sso`. Use GitHub CLI when setting
up Actions. Both launchers resolve the checkout from any working directory:

```sh
./deploy.sh --help
./deploy.sh --action build
./deploy.sh --action preview --environment dev
./deploy.sh --server
./deploy.sh --terminal
```

On Windows, replace `./deploy.sh` with `deploy.bat`. Plain terminal invocation
runs the shared deployment flow; plain browser invocation opens its action menu
with Preview selected. Eligible routine defaults use Composer's 15-second timer;
a complete browser settings form uses 30 seconds. Editing pauses the timer.
Passwords, browser account/apply decisions, and destructive changes require
explicit input. CI has no countdown and requires `--action deploy --approve`.
Build, setup, preview, status and report do not apply the application stack.
Setup can authorize state-bucket creation and GitHub setup independently.

The single target is the website plus its API. There is no separate admin/index
or discovery stack. `dev`, `qa`, `prod` and validated custom environments are
supported. Production support is explicitly enabled in this application's
manifest; isolated Composer test hosts retain their production restriction.

## Hostname and ownership

Supply the **full website hostname**, for example `cards.example.com`; it is not
a base domain from which this application invents another hostname. Use `none`
for the generated CloudFront URL. The manifest's labeled fields are available in
both wizard interfaces:

| Flag | Value |
| --- | --- |
| `--domain` | Full hostname or `none` |
| `--certificatearn` | Existing ISSUED ACM certificate in us-east-1 in the deployment account, or `none` |
| `--zoneid` | Existing public Route53 hosted zone ID, or `none` for manual DNS |

A custom hostname requires its certificate. The operator owns certificate
issuance/validation and the existing public DNS zone. With manual DNS, point the
hostname to the stack's CloudFront domain before expecting readiness. Pulumi
validates input structure before registering any resources; AWS verifies the
certificate and zone at deployment. No certificate or zone is silently created.

Existing resource logical names and protections are preserved. The Pulumi
project remains `card-designer`; stacks remain `<project>-<environment>`.
The new `deploymentIdentity` output verifies the selected project/environment.
The Lambda reads Composer's `archive` config; the legacy archive path remains a
fallback for existing direct Pulumi users. CamelCase `certificateArn`/`zoneId`
and `project`/`environment` remain readable when the new keys are absent.
Explicit `none` clears an old hostname/certificate/zone value.

## Builds and approval

Composer runs the asset builder before Lambda compilation. `dist/deploy/` holds
the Lambda ZIP and private build receipts. The fingerprint includes frontend and
drawing source, lockfiles, catalog, builder, Go dependencies/toolchain and
Node/npm versions. Cached files are reused only when their content digests match.
Use `--force-rebuild` to rebuild both outputs or `--action clean-cache` to clear
receipts. Each deployment still obtains a refreshed Pulumi preview.

The approval binds the saved plan, Lambda archive, source/configuration and
prepared website artifacts. A changed or missing file stops apply and requires a
fresh preview. Non-destructive terminal plans use the shared approval timer;
browser apply is explicit. Replacements/deletions require typed review and are
rejected by noninteractive deployment. Protected data, tokens and user pool
resources retain their existing protection settings.

After apply, `GET <siteUrl>/api/config` must return 200 within the bounded readiness
probe. Failure is reported as applied with readiness failed; it does not pretend
to roll back the stack or establish end-to-end acceptance. `--action status`
reads allowlisted outputs; `--action report` produces scoped artifact evidence,
and `--refresh-report` requests supported read-only verification.

## Saved records and recovery

Composer owns authoritative versioned records under
`~/.infra-composer/<project>/environments/<environment>/`. Repository `.deploy/`
contains checkout identity, local settings, encrypted credentials and private
reports. Both it and `dist/` are ignored. Explicit arguments override compatible
saved settings; account/region/state changes require a deliberate separate scope.
Do not copy/rename Composer vaults to another checkout: their encryption is bound
to their path. Preserve the state bucket/prefix, project, environment and original
Pulumi passphrase. S3 encryption, Pulumi secrets encryption and the local vault
are different controls; IAM must protect all state contents.

Existing settings from the previous Card Designer wizard need one offline
conversion, while no deployment process is running:

```sh
./deploy.sh migrate
./deploy.sh --action preview --environment dev --adopt-legacy-stack
```

The converter reads the original `.deploy/identity.json` and central
`~/.infra-composer/<project>/<environment>/settings.json` records. Back up these
and `.deploy/<environment>/vault.json` first. On a fresh checkout, restore those
original records before conversion. It validates all environments before writing,
preserves project/account/region/state URL, copies site inputs and existing stack
initialization receipts, and leaves every original record/vault intact. It refuses
incomplete or conflicting records. Identical partial writes are safe to retry.
It neither contacts AWS nor claims a successful cloud migration.

On first Composer unlock, choose its local vault password and provide the
**original Pulumi passphrase**. The converter does not decrypt or relabel the old
vault. Keep the original vault and its unlock password as recovery material; if
the Pulumi passphrase is only available inside it, recover it through the old
wizard before continuing. Never substitute a new stack passphrase. The first
preview of an existing stack needs deliberate `--adopt-legacy-stack` review
because old state has no `deploymentIdentity` output. Verify the target and
preview, then deploy with the same review flag; successful apply records the
identity for subsequent runs. A missing initialized stack stops for recovery.

After interruption, rerun the same command after fixing the reported problem.
Do not delete locks until the previous writer is confirmed stopped. Restore
missing state through S3 version history/Pulumi recovery, not stack recreation.
The old revision and untouched original records are rollback material before
apply. After apply, review a fresh preview before changing tooling versions; cloud
changes are not transactional rollback. Back up new central records, local vault
and their passwords as well as the S3 state location.

The adapter accepts the previous positional actions and aliases: `preview`,
`deploy`, `build`, `status`, `recover` (status), `reconfigure` (setup with
reconfiguration), `--state`, `--noninteractive`, `--yes`, `--certificate-arn` and
`--zone-id`. New scripts should use the shared flags. Use
`--export-settings PATH` / `--import-settings PATH` for Composer's allowlisted
nonsecret interchange; legacy vaults are never imported as settings.

## GitHub Actions

`deploy.yml` calls the same launcher and pinned CLI, with protected environments
named `dev`, `qa`, and `prod`, OIDC and serialized environment deployments. The
wizard offers GitHub setup; skip it for local-only deployment, or run:

```sh
./deploy.sh --action github-setup --environment dev --github-role arn:aws:iam::ACCOUNT:role/EXISTING_DEPLOY_ROLE
```

This consumer uses the **existing role/provider** integration. The operator must
create/review the OIDC provider and a least-privilege deployment role first. Its
trust must constrain audience to `sts.amazonaws.com` and subject to
`repo:lambdawalker/site.card-designer:environment:dev` (matching the environment).
The role needs only this stack's resources and its dedicated state prefix. The
wizard reconciles managed environment settings after explicit authorization,
preserving unrelated settings and protections. This manifest does not claim
automatic IAM role/policy creation. Never store AWS access keys or SSO tokens.

| Environment variable/secret | Content |
| --- | --- |
| `BENCH_PROJECT`, `BENCH_ACCOUNT`, `BENCH_REGION` | Exact deployment identity |
| `BENCH_BACKEND` | Existing S3 state URL with dedicated prefix |
| `BENCH_ROLE_ARN` | Reviewed OIDC deployment role |
| `BENCH_ENVIRONMENT` | Matching environment, reconciled by Composer |
| `BENCH_DOMAIN`, `BENCH_CERTIFICATE_ARN`, `BENCH_ZONE_ID` | Explicit reviewed site settings; use `none` when inapplicable |
| `PULUMI_CONFIG_PASSPHRASE` (secret) | Original stack passphrase |

`BENCH_` names are the shared CLI protocol, even for this website. Reconcile
existing Actions settings before using the migrated workflow; old `DEPLOY_*`
variables are not silently substituted. Require reviewers/branch restrictions for
sensitive environments. Run manual Preview first, inspect its report, then select
Deploy. The workflow validates mandatory inputs and passes approval only for
Deploy; it uploads no vault or raw state. No workflow was dispatched for this
migration. Existing stacks must finish interactive identity adoption first.

## Verification and remaining acceptance

Run the Go tests/vet in `backend`, `deploy`, `infra`; build from the normal launcher;
run a second build to verify cache reuse. CI tests both POSIX and Windows launchers
from a different working directory and runs the existing browser rendering suite.
The migration tests cover preserved scopes/vaults, conflicting/incomplete records
and retry. Pulumi mocks check authentication, private storage and API contracts.

Live AWS acceptance is separate and has not been performed: verify HTTPS/DNS,
Cognito sign-in/out, two-user design isolation, save/reload, browser sample download,
read-only token access/revocation, portrait package caching, bucket privacy and JSON
API errors. See the [requirement matrix](../docs/deployment/composer-migration.md).
