# Repository instructions for agents

These instructions apply to the entire repository unless a more specific
`AGENTS.md` exists in a subdirectory.

Go 1.26+ proxy server providing OpenAI/Gemini/Claude/Codex compatible APIs with OAuth and round-robin load balancing.

## Repository
- GitHub: https://github.com/router-for-me/CLIProxyAPI

## Commands
```bash
go build -o cli-proxy-api ./cmd/server # Build
go run ./cmd/server # Run dev server
go test ./... # Run all tests
go test -v -run TestName ./path/to/pkg # Run single test
go build -o test-output ./cmd/server && rm test-output # Verify compile (REQUIRED after changes)
```
- Common flags: `--config <path>`, `--tui`, `--standalone`, `--local-model`, `--no-browser`, `--oauth-callback-port <port>`

## Config
- Default config: `config.yaml` (template: `config.example.yaml`)
- `.env` is auto-loaded from the working directory
- Auth material defaults under `auths/`
- Storage backends: file-based default; optional Postgres/git/object store (`PGSTORE_*`, `GITSTORE_*`, `OBJECTSTORE_*`)

## Architecture
- `cmd/server/` — Server entrypoint
- `internal/api/` — Gin HTTP API (routes, middleware, modules)
- `internal/api/modules/amp/` — Amp integration (Amp-style routes + reverse proxy)
- `internal/thinking/` — Main thinking/reasoning pipeline. `ApplyThinking()` (apply.go) parses suffixes (`suffix.go`, suffix overrides body), normalizes config to canonical `ThinkingConfig` (`types.go`), normalizes and validates centrally (`validate.go`/`convert.go`), then applies provider-specific output via `ProviderApplier`. Do not break this "canonical representation → per-provider translation" architecture.
- `internal/runtime/executor/` — Per-provider runtime executors (incl. Codex WebSocket)
- `internal/translator/` — Provider protocol translators (and shared `common`)
- `internal/registry/` — Model registry + remote updater (`StartModelsUpdater`); `--local-model` disables remote updates
- `internal/store/` — Storage implementations and secret resolution
- `internal/managementasset/` — Config snapshots and management assets
- `internal/cache/` — Request signature caching
- `internal/watcher/` — Config hot-reload and watchers
- `internal/wsrelay/` — WebSocket relay sessions
- `internal/usage/` — Usage and token accounting
- `internal/tui/` — Bubbletea terminal UI (`--tui`, `--standalone`)
- `sdk/cliproxy/` — Embeddable SDK entry (service/builder/watchers/pipeline)
- `test/` — Cross-module integration tests

## Code Conventions
- Keep changes small and simple (KISS)
- Comments in English only
- If editing code that already contains non-English comments, translate them to English (don’t add new non-English comments)
- For user-visible strings, keep the existing language used in that file/area
- New Markdown docs should be in English unless the file is explicitly language-specific (e.g. `README_CN.md`)
- As a rule, do not make standalone changes to `internal/translator/`. You may modify it only as part of broader changes elsewhere.
- If a task requires changing only `internal/translator/`, run `gh repo view --json viewerPermission -q .viewerPermission` to confirm you have `WRITE`, `MAINTAIN`, or `ADMIN`. If you do, you may proceed; otherwise, file a GitHub issue including the goal, rationale, and the intended implementation code, then stop further work.
- `internal/runtime/executor/` should contain executors and their unit tests only. Place any helper/supporting files under `internal/runtime/executor/helps/`.
- Follow `gofmt`; keep imports goimports-style; wrap errors with context where helpful
- Do not use `log.Fatal`/`log.Fatalf` (terminates the process); prefer returning errors and logging via logrus
- Shadowed variables: use method suffix (`errStart := server.Start()`)
- Wrap defer errors: `defer func() { if err := f.Close(); err != nil { log.Errorf(...) } }()`
- Use logrus structured logging; avoid leaking secrets/tokens in logs
- Avoid panics in HTTP handlers; prefer logged errors and meaningful HTTP status codes
- Timeouts are allowed only during credential acquisition; after an upstream connection is established, do not set timeouts for any subsequent network behavior. Intentional exceptions that must remain allowed are the Codex websocket liveness deadlines in `internal/runtime/executor/codex_websockets_executor.go`, the wsrelay session deadlines in `internal/wsrelay/session.go`, the management APICall timeout in `internal/api/handlers/management/api_tools.go`, and the `cmd/fetch_antigravity_models` utility timeouts

## Remotes

- `origin` — `https://github.com/MrLinda/CLIProxyAPI.git` (personal fork)
- `upstream` — `https://github.com/router-for-me/CLIProxyAPI.git` (official)
- `kogeki` — `https://github.com/kogekiplay/CLIProxyAPI.git` (reference fork, base for `custom`)

## Branch structure

- `main` — exact mirror of `upstream/main`. No personal commits.
  Tracking: `upstream/main` (fetch/merge), push remote: `origin`.
  Sync: `git reset --hard upstream/main && git push origin main --force-with-lease`.
- `custom` — default branch on GitHub; long-lived maintenance branch based on `kogeki/main`, with `upstream/main` merged in periodically.
  Holds personal commits via cherry-pick (never rebase this branch).
  Push: `git push origin custom` (no force).
- `feature/*` — short-lived branches for PRs. Baseline is the PR target's default branch.

## Feature branch baseline selection

- PR to `router-for-me/CLIProxyAPI:main` → start from `upstream/main`.
- PR to `kogekiplay/CLIProxyAPI:main` → start from `kogeki/main`.
- PR to `kogekiplay/CLIProxyAPI:dev` → start from `kogeki/dev`.

Only cherry-pick the required feature commits. Do not move unrelated branch history into a PR.

Because the personal fork's default branch is now `custom`, GitHub may
auto-select `custom` as the base when creating a PR within the fork. When
opening a PR to the official upstream or to `kogekiplay`, always explicitly
set the correct base branch (`upstream/main`, `kogeki/main`, or `kogeki/dev`)
instead of relying on the auto-selected default.

## Sync workflow

See `SYNC.md` for the manual sync workflow.

## Backup branches

Before any destructive operation (reset, rebase, force-push), create a local backup:

```bash
git branch backup/pre-reorg-YYYYMMDD[-HHMMSS] main
git tag backup/<description>-pre-reorg-YYYYMMDD[-HHMMSS] <ref>
```

Check for name collisions with `git show-ref --verify --quiet` first.
Use a timestamp suffix when the name already exists.

## Test baseline policy

The full test suite must not regress relative to the maintained branch
baseline.

On Windows, `internal/api` currently has known environment-dependent baseline
failures caused by:

- SQLite temporary database files remaining locked during cleanup;
- integration tests requiring unavailable Redis-backed services.

These failures may be accepted only when all of the following are true:

- the same tests fail on the unmodified `kogeki/main` baseline;
- the failure names and error signatures are unchanged;
- no additional test fails after merging or cherry-picking;
- all packages directly affected by the current changes pass targeted tests;
- the application build succeeds;
- the baseline comparison logs are preserved and reported.

Do not use this exception for new failures, different assertions, panics,
compile errors, race failures, or failures in packages changed by the current
work.

## Allowed untracked content

After any git operation, the only allowed visible untracked entry in
`git status --short` is:

```
?? pr-body.md
```

- `pr-body.md` is a local PR description draft kept for convenience.
- `docker-build/` is intentionally git-ignored and will not appear in
  `git status --short`. It contains local Docker build artifacts (`.tar`
  files). Never delete, move, stage, commit, or `git clean` them.
  Never commit the `.tar` files in this directory to any branch.
- Any other untracked, modified, staged, or conflicted entry must be reported
  and investigated before proceeding with destructive operations.

## Formatting rules

Do not use `gofmt -w .` — it may modify unrelated files.

Only when Go files have been manually modified (e.g. after merge conflict
resolution), format the specific files:

```bash
git diff --name-only --diff-filter=ACM -- '*.go'
gofmt -w <file1> <file2> ...
```

After formatting, rebuild and retest:

```bash
go build -o test-output ./cmd/server
rm -f test-output
go test ./...
```

If no Go files were manually modified, do not run gofmt.

## Conflict handling

When a merge or cherry-pick produces conflicts:

1. Stop immediately.
2. Output:
   ```bash
   git status --short
   git diff --name-only --diff-filter=U
   ```
3. List the conflicted files and the modules they belong to.
4. Do not:
   - choose `ours` or `theirs`;
   - modify business logic;
   - skip the conflicted commit;
   - automatically `git merge --abort`.
5. If abort is needed, explain the planned command first and wait for
   confirmation.

## Build and test rules

Standard validation sequence:

```bash
go build -o test-output ./cmd/server
rm -f test-output
go test ./...
```

Rules:
- Stop on build failure.
- Stop on any new test failure relative to the baseline.
- Packages directly modified by the current changes must pass targeted tests.
- Do not push `custom` when build fails or new regressions appear.
- Known platform-dependent baseline failures must be verified against the
  unmodified baseline before being accepted (see Test baseline policy above).

## Document scope

- `AGENTS.md` and `SYNC.md` belong to the `custom` branch.
- They must never be committed to the `main` (pure mirror) branch.
- Before creating or updating these documents, check whether they already
  exist. Do not blindly overwrite existing content.
- Do not create automated GitHub Actions sync workflows unless explicitly
  requested.

## Reorganization commit records

The following commits and references were established during the initial
repository reorganization:

| Item | SHA | Description |
| --- | --- | --- |
| Original think-tag-parsing commit | `1c54269b7302797778ccbe80824d48e03e450b0a` | On `feat/think-tag-parsing` branch |
| Cherry-picked in custom | `2a00d9fbd1aa5a8962076e75526d7bc38658b672` | Same patch, new SHA after cherry-pick |
| Stable patch-id | `7959379bf68089843849ce6359ebe18bc2ad5f61` | Both commits produce this patch-id |
| Upstream merge commit | `928b835a31022c2bac933feef913fb863084fa7c` | `git merge upstream/main --no-edit` |
| AGENTS.md initial commit | `10842095` | First maintenance policy commit |
| SYNC.md initial commit | `a032ab63` | First sync workflow commit |
| Backup branch | `backup/pre-reorg-20260720` → `fc777089cabf44453dcc37ef7f73803562b1a1b6` | `main` before reset |
| Backup tag | `backup/feat-think-tag-parsing-pre-reorg` → `1c54269b7302797778ccbe80824d48e03e450b0a` | `feat/think-tag-parsing` before operations |
| pr-body.md external backup | `..\CLIProxyAPI-pre-reorg-backup-20260720-214219\pr-body.md` | Backup copy outside repository |

Verification rules for cherry-picked patches:
- `git show 1c54269b | git patch-id --stable` should match
  `git show 2a00d9fb | git patch-id --stable`.
- If patch-id differs, use `git range-diff 1c54269b^! 2a00d9fb^!` to compare
  the actual delta. Do not use a plain tree diff.

These SHAs are historical records. Future `custom` HEAD changes are normal.
Do not use these SHAs to block normal updates.

### Reorganization test matrix

Three-state comparison performed at reorganization time:

| State | internal/api | internal/runtime/executor |
| --- | --- | --- |
| kogeki-main (`fc777089`) | 36 failures (baseline) | 1 flaky failure |
| merged-before-patch (`928b835a`) | same 36 failures | passed |
| custom-after-patch (`2a00d9fb`) | same 36 failures | passed |

The merge and cherry-pick introduced **no new test failures** relative to the
kogeki/main baseline. Build succeeded. All 20+ affected Go packages passed
targeted tests.

Diagnosis logs preserved at:
```
E:\Code\personal\CLIProxyAPI-test-matrix-20260720-222036\
```

This path is specific to the current Windows workspace. Future agents should
not treat its absence as an anomaly. New baseline diagnostics should generate
and report a new log path.

### Initial custom image release

The following records document the first custom image release.

Version tag: `v7.2.91-custom.1`

Released commit: `f5f72dc7458a62b93e8d6aea28d2cbd3ed423d8a`

Top-level image digest:
`sha256:dff33cb05f604faa701f575102eb8e648b2c991b7e98d17f0f101d937c17150f`

Three GHCR image tags (verified identical digest):
- `ghcr.io/mrlinda/cli-proxy-api:custom`
- `ghcr.io/mrlinda/cli-proxy-api:v7.2.91-custom.1`
- `ghcr.io/mrlinda/cli-proxy-api:sha-f5f72dc7458a62b93e8d6aea28d2cbd3ed423d8a`

The validate and publish jobs both succeeded. The GitHub Release workflow also
ran successfully for this tag.

The initial custom tag push also triggered the legacy Docker Hub workflow,
which failed due to missing Docker Hub credentials in this fork. A subsequent
commit, `e51093f73e668afc02963e61b43d62109ef57408`, excluded `v*-custom.*` tags
from that workflow. The workflow was later fully removed from the repository;
custom images are now published exclusively by `custom-ci-image`.
Therefore, the most recently released image may
legitimately reference an earlier commit (`f5f72dc7`) than the current
`custom` branch HEAD.

These SHAs and digests are historical records of the initial custom release.
Future releases will use different version tags, commits, and digests.

### Default branch and feature branch cleanup

The following changes were applied after the initial release:

- GitHub default branch was changed from `main` to `custom`.
- `origin/HEAD` updated to point to `origin/custom`.
- `feat/think-tag-parsing` branch (local and `origin`) was deleted after
  verifying patch-id equivalence (`7959379bf68089843849ce6359ebe18bc2ad5f61`)
  between the original commit (`1c54269b`) and its cherry-pick in `custom`
  (`2a00d9fb`).
- No open PRs were associated with the deleted branch.
- The original commit is preserved by the backup tag
  `backup/feat-think-tag-parsing-pre-reorg` (→ `1c54269b`).
- Deleting the branch does not remove the functionality; `custom` already
  contains the equivalent patch.

These are historical records. Future cleanup operations may differ.

### Inherited workflow cleanup

The following inherited GitHub Actions workflows were removed from the
repository after the default branch was changed to `custom`:

- `.github/workflows/agents-md-guard.yml` — closed PRs modifying `AGENTS.md`;
  unnecessary in a personal fork with no open contributions.
- `.github/workflows/auto-retarget-main-pr-to-dev.yml` — retargeted `main` PRs
  to `dev`; `dev` no longer exists as a remote branch.
- `.github/workflows/docker-image.yml` — published to Docker Hub; the fork
  now publishes only to GHCR via `custom-ci-image`.
- `.github/workflows/pr-test-build.yml` — built (but did not test) any PR;
  `custom-ci-image` covers PRs to `custom` with build and full test.
- `.github/workflows/pr-path-guard.yml` — blocked `internal/translator/`
  changes in PRs; translator modifications now follow normal review and test
  processes.

Removing these workflows did not affect GitHub Release publishing, GHCR
image publishing, or Dependabot Dependency Graph functionality.

These are historical records. Future cleanup operations may differ.

## Mandatory stop conditions

Operation must stop immediately when any of the following occurs:

- Merge or cherry-pick conflict.
- Build failure.
- New test failure relative to the unmodified baseline.
- Targeted test failure in packages directly modified by the current work.
- `git push origin main --force-with-lease` is rejected.
- Remote ownership or branch ancestry does not match documented structure.
- Operation would overwrite an existing branch, tag, or file without explicit
  authorization.
- Working tree contains unexpected modified, staged, or untracked content
  (beyond `pr-body.md` and the git-ignored `docker-build/`).
- Determining which commits belong to personal work requires guessing.
- Local or remote `custom` would be overwritten.
- An existing `AGENTS.md` or `SYNC.md` would be blindly overwritten.
- `git clean` or force-push without authorization is required to proceed.

## Custom release policy

### Validation-only pushes

Ordinary pushes to `custom` run validation only (build and test). They do not
publish or replace container images. The `custom` branch HEAD may be newer
than the commit represented by the `ghcr.io/mrlinda/cli-proxy-api:custom`
image tag. This is expected when validated commits have not yet received a
version release tag.

### Version tag triggers

Only annotated Git tags matching `v*-custom.*` trigger image publication.
Examples: `v7.2.91-custom.1`, `v7.2.91-custom.2`.

Tag format meaning:
- `vX.Y.Z` is the official upstream version baseline.
- `custom.N` is the Nth custom release on top of that baseline.

After the official baseline updates, the custom sequence may restart at 1
(e.g. `v7.2.92-custom.1`).

### Three image tags

A successful version tag workflow builds the Docker image once and publishes
three tags pointing to the same digest:

| Tag | Semantics | Mutable? |
| --- | --- | --- |
| `ghcr.io/mrlinda/cli-proxy-api:custom` | Latest released custom version | Yes (rolling) |
| `ghcr.io/mrlinda/cli-proxy-api:<version>` | Fixed version identifier | No |
| `ghcr.io/mrlinda/cli-proxy-api:sha-<commit>` | Immutable commit pointer | No |

After publication, verify all three tags share the same top-level digest.

### Pre-release checklist

1. Confirm current branch is `custom` and matches `origin/custom`.
2. Confirm `custom` branch validate workflow succeeded and publish was skipped.
3. Confirm the target version tag does not exist locally or on `origin`.
4. Confirm the tag points to the intended commit.
5. Create an annotated tag:
   ```bash
   git tag -a <version-tag> <commit-sha> -m "Release <version-tag>"
   git push origin refs/tags/<version-tag>
   ```

### Post-release verification

- Validate job succeeded.
- Publish job succeeded.
- Release workflow completed successfully (GitHub Release + multi-platform
  packages).
- Legacy Docker Hub workflow was excluded from custom tags and
  later removed from the repository.
- All three GHCR tags reference the same top-level digest.
- `VERSION` build arg equals the Git tag.
- `COMMIT` build arg equals the tag's peeled commit.

### Prohibited actions

- Moving, deleting, or overwriting a published version tag.
- Reusing a version number with a different commit.
- Force-pushing any tag.
- Deleting a failed-version tag and re-releasing under the same name.
- Auto-incrementing the next version number.
- Publishing a `latest` tag.
- Confusing an ordinary `custom` branch push with a release.
- Running `publish` when `validate` fails.
- Logging into GHCR or publishing images from a pull request.
- Using PAT, password, or private key instead of `GITHUB_TOKEN`.
- Auto-modifying GHCR package visibility.
- Auto-deploying to production servers.

### Legacy Docker Hub workflow

The inherited Docker Hub workflow was removed after the repository adopted
the GHCR-only custom release process. Custom images are published exclusively
by `custom-ci-image`. The personal fork no longer publishes to
`eceasy/cli-proxy-api` and no longer requires `DOCKERHUB_USERNAME` or
`DOCKERHUB_TOKEN`.

### Existing release workflow

The existing `.github/workflows/release.yaml` responds to all tags (including
`v*-custom.*`) and generates GitHub Releases with multi-platform packages.
This is intentional: the same tag may trigger both `custom-ci-image` (GHCR)
and `release` (GitHub Release).
