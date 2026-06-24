# cliwright templates

Copy-in skeletons for a **generated** CLI. They are not part of cliwright's own runtime —
the build process drops them into the new project and fills the placeholders.

## Placeholder convention

Replace these literal tokens during generation:

| Token | Meaning | Example |
|---|---|---|
| `__BINARY__` | command users type | `stripe` |
| `__MODULE__` | Go module path | `github.com/me/stripe-cli` |
| `__OWNER__` | GitHub owner/org | `me` |
| `__REPO__` | repo name | `stripe-cli` |

GoReleaser's own `{{ .Version }}` / `{{ .ModulePath }}` etc. are GoReleaser templating —
leave those intact.

## Where each file goes

| Template | Destination in the generated CLI |
|---|---|
| `Makefile` | `./Makefile` (defines `make verify`, the acceptance gate) |
| `goreleaser.yaml` | `./.goreleaser.yaml` |
| `ci.yml` | `./.github/workflows/ci.yml` |
| `cover-check.sh` `dod-check.sh` `spec-check.sh` `judge.sh` | `./scripts/` (chmod +x) |
| `resource.api.go.tmpl` | `internal/api/<resource>.go` (one per resource) |
| `resource.cmd.go.tmpl` | `commands/<resource>.go` (one per resource) |
| `resource_test.go.tmpl` | `internal/api/<resource>_test.go` |
| `api-manifest.example.json` | `./api-manifest.json` (the §11 determinism anchor) |

## The gate

`make verify` = `check` + `spec-check` + `cover-check` + `dod-check.sh` + `judge.sh`.
It is the loop's exit condition (GOAL.md §12): the `/goal` completion promise may fire
only when `make verify` exits 0. `dod-check.sh` is deterministic (one check per atomic
DoD item); `judge.sh` is the single LLM-scored gate for the few subjective items and is
skippable in CI only via `CLIWRIGHT_SKIP_JUDGE=1` (logged, never silent).
