# Changelog

All notable changes to cliwright are documented here. Format: [Keep a Changelog](https://keepachangelog.com); versioning: [SemVer](https://semver.org).

## [Unreleased]

## [0.6.1] — 2026-07-19

### Fixed
- **Generated CLIs shipped without a live MkDocs site.** `docs.yml` was copied inconsistently and
  GitHub Pages was never enabled, so `https://<owner>.github.io/<repo>/` 404'd even though
  `mkdocs gh-deploy` had pushed `gh-pages`. GOAL.md §6 now marks `docs.yml` as non-optional and
  calls out that Pages must be enabled separately; §10 adds a **"Publishing the doc site"** runbook
  (dispatch the Docs workflow the first time, `gh api POST .../pages` to enable Pages on `gh-pages`,
  verify the URL returns 200), plus the reminder that a brand-new repo can silently fail to fire
  CI/Release/Docs on the initial push — confirm with `gh run list` and dispatch by hand.

### Added
- **`workflow_dispatch` on `templates/ci.yml`, `templates/release.yml`, and `templates/docs.yml`** —
  so a fresh repo that doesn't auto-fire on the first push (or tag) can be triggered manually with
  `gh workflow run <name> --ref <branch-or-tag>`. Previously only some workflows could be dispatched.

## [0.6.0] — 2026-07-13

### Added
- **Ready-to-use templates for the highest-reuse §3d patterns** — so an agent copies proven code
  instead of re-deriving it (the token savings), while staying flexible via documented adaptation
  notes (each template is generic/org-agnostic, gofmt-clean; the event-store compiles standalone):
  - `templates/sanitize.go` — terminal-escape sanitizer for API text in the human/table path.
  - `templates/write.go` — universal `--data`/`--set`/`--file`(+stdin) write flags → attributes map,
    with the JSON:API envelope + `--rel` as a documented adaptation.
  - `templates/store.cache.go` — offline-cache SQLite store (pull-only time-series flavor).
  - `templates/store.events.go` — event-store SQLite store (live-stream flavor: FTS5-or-LIKE search
    with automatic fallback, `(profile,topic,ts)` dedup, prune, + the warn-and-continue `Recorder`
    wiring note).
  §3d now points each of those patterns at its template.

## [0.5.0] — 2026-07-13

Distilled from a full audit of the 9-CLI fleet: fold the genuinely-reusable patterns each CLI evolved
back into the playbook — with the CONDITION under which to apply each, so cliwright knows *when*, not
just how. (Org-specific fleet infrastructure — the reusable CI/release workflows — deliberately stays
out of cliwright; it lives in the org's own AGENTS.md.)

### Added
- **§3d Conditional patterns catalog** — a trigger-keyed menu of ~18 proven patterns so cliwright
  applies each only when its condition holds (a chat CLI needs an event-store, a metrics CLI an
  offline cache, an accounting CLI neither): local **event-store** + `log`/`listen` (API pushes a
  stream / has no history endpoint) vs **offline cache** + `sync`/`history` (pull-only time-series);
  **spec-contract test** (manifest carries real paths); **binary integration tests** (`-tags
  integration`); **universal write flags** (`--data`/`--set`/`--file`); **multi-group / path-routed
  credentials**; **adopt-an-existing-typed-library** build-mode; **terminal-escape sanitization** of
  API text; **options structs + `Validate()`**; **redacting `slog`**; **`-coverpkg -count=1`**;
  **`smoke.yml`/`spec-sync.yml`** drift workflows; **batch pool**, **response cache**, **self-update**,
  **drop-in symlink**, **cmdtest harness**, **import-sibling-session**. §0 research now evaluates the
  triggers up front and records N/A in `DECISIONS.md`.

### Changed
- **Canonical root construction** (settles a three-way fleet drift): `init()` appends to a registrar
  queue and a `NewRootCmd(deps)` constructor drains it — thin `init()` registration **and** a testable
  no-mutable-global-root tree. Bans both mutating a package-level `rootCmd` and hand-writing
  `newXCmd()` per resource; `dod-check` should reject a directly-mutated global root.

### Fixed
- GOAL.md §1 prose still said `term.ReadPassword`; the template already ships raw-mode `readSecretRaw`.
  Corrected (canonical-mode `ReadPassword` caps at `MAX_CANON`=1024 and hangs on long pasted keys).

## [0.4.0] — 2026-07-13

### Fixed
- **CI/release/docs templates now read `go-version-file: go.mod`, not `go-version: stable`.** This
  was the root cause of a fleet-wide exposure: `stable` grabs the newest Go, so `govulncheck` runs
  against an already-patched stdlib and stays GREEN even when the module's declared toolchain floor
  is behind — a real stdlib CVE hides behind a passing CI. Reading the floor from `go.mod` makes CI
  test what users actually build. GOAL.md §6 previously recommended `stable`; it now prescribes
  `go-version-file` with the reasoning. Every CLI generated from here on is unaffected; existing
  generated CLIs should flip the same four lines.

### Added
- **`templates/release.yml`, `templates/docs.yml`, `templates/dependabot.yml`.** These three
  workflows were previously generated ad-hoc from GOAL.md prose, so they drifted per repo
  (goreleaser-action v6/v7, cosign-installer v3/v4) and Dependabot ended up missing entirely on
  some repos. Shipping them as templates makes the generated `.github/` deterministic. The
  `dependabot.yml` (weekly grouped gomod + github-actions) doubles as dependency-CVE early warning.
- A note in `templates/ci.yml` on when to add `-coverpkg=./...` (coverage that relies on
  cross-package integration tests under-reports without it).

## [0.3.3] — 2026-07-12

### Fixed
- **`templates/prompt.go` reads the hidden secret in raw mode.** `term.ReadPassword` reads in
  canonical terminal mode, capped at `MAX_CANON` (1024 bytes on macOS): pasting a longer secret
  (a ~970-char JWT) fills the line buffer and the terminal blocks until Ctrl-C — the "prompt hangs
  on a long key" bug. The template now reads via `readSecretRaw`/`scanSecretLine` (raw mode, no
  line-length limit; Ctrl-C cancels, Backspace edits, non-TTY pipes fall back to a line read), with
  `scanSecretLine` split out so the byte handling is unit-testable without a PTY. Surfaced on
  lemon-squeezy-cli's `auth login` (HTTP 401 from a truncated key) and fixed fleet-wide.

## [0.3.2] — 2026-07-11

### Added
- **install.sh is part of the standard** (GOAL §5/§9, `templates/install.sh`, DoD gate): every
  generated CLI ships a zero-infra, checksum-verifying `curl | sh` installer for macOS/Linux that
  discovers the release archive from `checksums.txt` — naming-agnostic across goreleaser's
  `_amd64`/`_x86_64` (and `arm64`/`aarch64`) conventions.
- **`commands/prompt.go` template** — `promptSecret` (`term.ReadPassword`, pipe fallback) and
  `promptLine`, so generated CLIs read input through the right helpers.

### Security
- **Secrets must be prompted hidden.** GOAL §1 now requires reading tokens/keys/passwords/OAuth
  codes via a hidden prompt, and `dod-check.sh` fails on any `fmt.Scan`/`Scanln`/`Scanf` call —
  which echo the secret in plaintext (into scrollback) and stall on long pastes. Two generated
  CLIs had shipped this (lemon-squeezy-cli, canvas-cli); both fixed.

## [0.3.1] — 2026-07-02

Hardening from a fleet-wide audit of the agent guard across the generated CLIs (canvas, alegra, n8n, tgctl, lemon-squeezy).

### Added
- **Mandatory agent-guard hook hardening** in GOAL.md §3b — a nine-point checklist, each item a real verified bypass or dead-config bug found in the audit: the PreToolUse hook is required (not just permission rules); the anchored ERE must include the `([^[:space:]]*/)?` path prefix and a separator-accepting trailing boundary; the no-jq fallback must flatten JSON punctuation (else it fails open) and its test must build a strict `PATH` that truly hides `jq`; nothing that mutates remote state may classify as read/local (verb collisions + annotation gaps); enumerate cobra aliases; verify the raw-api escape's real syntax; Claude permission rules are literal prefixes, not regex; emit real Codex/OpenCode schemas; and ship the hook execution battery as tests.
- **`dod-check.sh` gates** for the hook generator, the execution-battery test file, the path-prefix hardening, and the no-jq JSON flattening.

## [0.3.0] — 2026-06-30

Hardening from the tgctl & lemon-squeezy build post-mortems: API completeness, per-tool ergonomics, a leaner gate, CI/Windows fixes, and a self-contained skill.

### Added
- **Method-enumeration step + completeness gate** (closes the narrow-surface root cause). The spec-check gate only ever proved CLI ⊆ manifest (consistency), never manifest == full API (completeness), so under-capture was invisible. GOAL.md §0 Step 1b now REQUIRES enumerating the complete method/endpoint set from a source (OpenAPI/Postman/`llms.txt`, else the docs' full method index or a community machine spec — Telegram → `ark0f/tg-bot-api`) BEFORE authoring the manifest, which must derive from that list, not model recall. A new `scripts/spec-completeness.sh` (template) compares the manifest's covered method count (`resources[].verbs` + `methods[]`) against the enumerated `api_method_total` and FAILS below ~90% unless a `coverage-waiver` is recorded in `DECISIONS.md`. Wired into `make verify` alongside `spec-check` (Makefile `verify`/`spec-completeness` targets). Manifest gains `api_method_total` + `api_method_source` (§0/§9/§11/§12).
- **Configurable multi-profile flag name** (per-tool). New manifest/TARGET field `profile_flag`/`profile_noun` (default `"profile"`) names the multi-profile selector so it reads naturally — `--bot` for Telegram, `--instance` for n8n, `--account` for accounting. `--profile` is kept as a HIDDEN alias for back-compat; GOAL.md §1/§3 carry the root.go wiring snippet, and the MCP exclusion (§3b) + guardrails now exclude the selector under both its configured name and the `--profile` alias (§0 TARGET, §1, §3, §3b).

### Changed
- **Judge decoupled from `make verify`.** The LLM judge was part of `make verify`, so it re-ran on every later CI/dev invocation — draining tokens, non-deterministic, failing in CI (no agent). Now `make verify` is the deterministic gate (CI/dev), `make judge` is the LLM gate, and `make accept` (= `verify` + `judge`) is the build-acceptance gate the `/goal` loop binds to. Updated the Makefile template, GOAL.md §12 + the completion-promise bindings, SKILL.md, `ralph.sh`, and the template README.
- **GOAL.md ships inside the skill** (`skills/cliwright/GOAL.md`), so a skill-only `npx skills add` install carries the playbook — no network fetch, no version skew, no dependency on another repo. All references updated to resolve under both plugin and skill-only installs.

### Fixed
- **CI hardened against recurring build traps.** The `ci.yml` template now builds golangci-lint/gosec/govulncheck from source with the job's Go (the prebuilt actions lagged the toolchain and choked on a go1.25 module — "configuration contains invalid elements"). GOAL.md test guidance now mandates deadlock-safe stdout capture (cobra `SetOut` / goroutine-drained `os.Pipe` — the Windows pipe-buffer hang) and path-portable tests (`filepath.Join`, `t.TempDir`). The fixes the per-build agents kept rediscovering are now baked into the template.

## [0.2.1] — 2026-06-24

Fixes a manifest bug that broke `/plugin install cliwright@cliwright` on 0.2.0, and adds validation so it can't regress.

### Fixed
- **Plugin install failed schema validation** (`agents: Invalid input`). `plugin.json` declared `agents: "./agents"`, but the manifest schema only accepts a `.md` file path (or list) for `agents` — not a directory. Removed the redundant `agents`/`commands`/`skills` directory pointers entirely; those conventionally-named directories are auto-discovered.
- **`marketplace.json` `$schema` pointed at a 404 URL** (`claude-code-plugin-marketplace.json`); corrected to the real `claude-code-marketplace.json` so editors validate it live.

### Added
- **Manifest validation** (`scripts/validate-plugin.py`) — JSON Schema validation against the vendored schemastore schemas in `.claude-plugin/schemas/`, plus structural checks that every referenced component resolves and carries the required frontmatter.
- Wired into a **pre-commit hook** (`.githooks/pre-commit`, enable via `scripts/install-hooks.sh`) and **CI** (`.github/workflows/validate.yml`), reusing the one validator so local and remote checks never drift.

## [0.2.0] — 2026-06-24

Hardening pass driven by a real dogfood (built `catapi-cli` from the playbook to a green gate) plus a comparison against four mature CLI codebases and the actual n8nctl `/goal` build transcript.

### Added
- **Multi-auth** — `Authenticator` interface with one implementation per method; the profile records the method; `auth login --method` (or per-method subcommands) selects. Closes the single-auth regression (§0/§1/§3).
- **User-controlled distribution** — `distribution_scope` TARGET field (`local-build` default). The agent never creates a remote repo, pushes, or releases on its own (§0/§4/§10/Guardrails).
- **Existing-CLI detection** — surface a first-party/competing CLI and the build-vs-adopt trade-off; the user decides, the agent never refuses to build (§0/Guardrails).
- **Determinism "adapt to the API" rules** — flexible types / profiles / CSV are defaults, dropped only by rule recorded in `DECISIONS.md` (§2).
- Flexible `Int`/`Bool` types + Money/Int rigor (NaN/Inf rejection, Int64-before-Float64) (§2).
- Optional, user-gated, explicitly-destructive **live smoke test** (Phase F2); `completions.sh` template; goreleaser `release:` block.

### Changed
- **Drop Viper** — config resolves via manual `firstNonEmpty(...)`, the real house pattern; env names are per-service, not a uniform `<BIN>_TOKEN` (§1).
- §3c match rule → **match by name + mandatory duplicate-skip** when no stable handle; "unchanged" = field whitelist (§3c).
- §3b annotation precision — ophis `Annotation*` constants → singular keys; no "write" key (write = `openWorldHint`).
- The gate is **`make verify`, not `make check`**, for every surface-touching change; coverage is a ratchet; annotate resources in the Phase E loop.
- Retry honors `Retry-After` + `OPTIONS`; rate-limiting has a no-quota-headers branch; CSV formula-injection guard (CWE-1236); config hardening (`0700/0600`/atomic) + profile/URL validation.
- Review discipline: expect ~50% false positives, verify findings against code **and comments**, record cited refusals; pre-sweep test fixtures before tightening a shared validator.

### Fixed
- `dod-check.sh` goreleaser check no longer soft-passes on failure (`|| true` removed).
- `spec-check.sh` now asserts each declared **verb**, not just resource presence.

## [0.1.0] — 2026-06-24

Initial release: the GOAL playbook, Claude Code plugin + cross-tool agent skill, `/cliwright:build-cli` command, `cli-builder` subagent, and the gate-runnable templates.
