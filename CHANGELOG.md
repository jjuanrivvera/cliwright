# Changelog

All notable changes to cliwright are documented here. Format: [Keep a Changelog](https://keepachangelog.com); versioning: [SemVer](https://semver.org).

## [Unreleased]

Two design fixes from the tgctl post-mortem (a hand-curated manifest wrapped only ~⅓ of the Telegram Bot API) and the `--profile` ergonomics review.

### Added
- **Method-enumeration step + completeness gate** (closes the narrow-surface root cause). The spec-check gate only ever proved CLI ⊆ manifest (consistency), never manifest == full API (completeness), so under-capture was invisible. GOAL.md §0 Step 1b now REQUIRES enumerating the complete method/endpoint set from a source (OpenAPI/Postman/`llms.txt`, else the docs' full method index or a community machine spec — Telegram → `ark0f/tg-bot-api`) BEFORE authoring the manifest, which must derive from that list, not model recall. A new `scripts/spec-completeness.sh` (template) compares the manifest's covered method count (`resources[].verbs` + `methods[]`) against the enumerated `api_method_total` and FAILS below ~90% unless a `coverage-waiver` is recorded in `DECISIONS.md`. Wired into `make verify` alongside `spec-check` (Makefile `verify`/`spec-completeness` targets). Manifest gains `api_method_total` + `api_method_source` (§0/§9/§11/§12).
- **Configurable multi-profile flag name** (per-tool). New manifest/TARGET field `profile_flag`/`profile_noun` (default `"profile"`) names the multi-profile selector so it reads naturally — `--bot` for Telegram, `--instance` for n8n, `--account` for accounting. `--profile` is kept as a HIDDEN alias for back-compat; GOAL.md §1/§3 carry the root.go wiring snippet, and the MCP exclusion (§3b) + guardrails now exclude the selector under both its configured name and the `--profile` alias (§0 TARGET, §1, §3, §3b).

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
