# Syncing with the official upstream

This repository tracks the official upstream
`router-for-me/CLIProxyAPI` and the reference fork
`kogekiplay/CLIProxyAPI`.

## Branches

- `main` — exact mirror of `upstream/main`. No personal commits.
- `custom` — long-lived maintenance branch based on `kogeki/main`.
  It receives official updates by merging `upstream/main` and contains
  locally maintained changes.
- `feature/*` — short-lived branches created from the actual PR target.

## Choose the feature branch baseline

- PR to `router-for-me/CLIProxyAPI:main` → start from `upstream/main`.
- PR to `kogekiplay/CLIProxyAPI:main` → start from `kogeki/main`.
- PR to `kogekiplay/CLIProxyAPI:dev` → start from `kogeki/dev`.

Only cherry-pick the required feature commits. Do not move unrelated branch
history into a PR.

## Manual sync workflow

```bash
# Refresh all remote-tracking references, including origin/main
git fetch --all --prune

# Keep main as an exact mirror of the official upstream
git switch main
git reset --hard upstream/main
git push origin main --force-with-lease

# Merge official updates into the maintained custom branch
git switch custom
git merge upstream/main --no-edit

# Validate before publishing custom
go build -o test-output ./cmd/server
rm -f test-output
go test ./...

# Push only after validation succeeds
git push origin custom
```

## Rules

- Never delete remote branches as part of synchronization.
- Never merge `kogeki/main` or `kogeki/dev` into `main`.
- Do not rebase the long-lived `custom` branch.
- Use `--force-with-lease` only for the mirrored `main` branch.
- Do not push `custom` when build or tests fail.
- Create each PR branch from the branch that will actually receive the PR.
- When the full test suite has a documented platform-dependent baseline
  failure, compare it against the unmodified baseline and do not push if any
  new failure appears.
