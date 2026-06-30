---
name: cliwright
description: |
  Forge a complete, production-grade, agent-ready command-line tool (CLI) for any REST/HTTP API. Use when the user wants to build, scaffold, generate, or create a CLI; wrap an API in a command-line tool; turn an API into a CLI; or make a `gh`-style tool for a service. Produces a Go + Cobra + GoReleaser binary with OS-keyring auth, named profiles, an MCP server, an agent guard, CI/CD, and packaging — and drives the build to completion against a deterministic acceptance gate (`make verify`) so every quality criterion is provably met, not merely asserted. Triggers: "build a CLI for X", "scaffold a CLI", "wrap the X API in a CLI", "make a command-line tool for X", "API to CLI", "create a gh-style CLI".
version: 0.2.0
homepage: https://github.com/jjuanrivvera/cliwright
license: MIT
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, WebFetch, WebSearch
metadata: {"openclaw":{"category":"developer-tools","emoji":"🛠️"}}
---

# cliwright — a wright of CLIs

This skill turns any HTTP API into a polished, distributable CLI built to a single
high standard, fully specified in **[GOAL.md](GOAL.md)** (the canonical playbook).
The playbook is self-contained: it does **not** depend on reading any other repository.

## How to run it

1. **Read the playbook.** Load `GOAL.md` (in this skill's directory — it ships next to this
   SKILL.md, so the same path resolves under both a plugin and a skill-only install). It is
   the complete brief — architecture, the
   non-negotiable standard, the meta-command set, the MCP/agent surface, distribution,
   CI/CD, packaging, reference skeletons, the determinism rules, and the acceptance gate.
2. **Fill the TARGET API block** at the top of the playbook. Everything about the API
   itself (auth, base URL, pagination, rate-limit headers, resources, fields, special
   ops, JSON quirks) is research, not a question — fetch the docs/OpenAPI and determine it.
   Only ask the user what the web cannot know (their instance/host, scope preferences,
   distribution identity), batched into ≤4 questions; skip questions already answered.
3. **Execute the phases in order** (Scaffold → Client core → Cross-cutting UX → Meta
   commands → Resource loop → Agent surface → Beyond-the-API → Tests & gates → Docs →
   Distribution → Packaging), committing per phase.
4. **Gate on `make accept`.** The build is done only when the acceptance gate passes:
   `make verify` (the **deterministic** gate — `make check` + coverage + `spec-check` (CLI
   surface ⊆ the manifest) + `spec-completeness` (manifest covers ≥ ~90% of the enumerated
   API, so a memory-authored manifest can't wrap a fraction of it unnoticed) + the atomic
   Definition-of-Done checklist) **plus** the LLM `judge` rubric for the few subjective items.
   `make accept` = `verify` + `judge`; the judge spends tokens and needs an agent, so CI and
   routine runs use the cheaper `make verify`. See the "Acceptance gate", "Determinism rules",
   and §0 method-enumeration sections in GOAL.md.

## Driving it to completion (the loop)

This skill carries the **spec and the gate**; it relies on the runtime's native `/goal`
loop as the **engine**. Run the build under `/goal` (Claude Code and Codex both ship it),
and bind the loop's completion promise to the gate:

> The completion promise may be emitted **only** when `make accept` exits `0`.
> Never emit a false promise to escape the loop — if the gate fails, keep iterating.

If `/goal` is unavailable, fall back to the bundled `scripts/ralph.sh` (a portable
`while ! make accept; do <agent> continue; done` loop).

## What it produces

A tagged, installable CLI: Homebrew/Scoop + deb/rpm/apk, signed releases (cosign + SBOM),
generated command-reference docs, an `mcp` server + `agent guard`, and — if requested —
its own Claude Code plugin + cross-tool agent skill.

See `references/` for deep dives extracted from the playbook, and `templates/` for the
copy-in skeletons (generic core, Makefile with the `verify` target, GoReleaser, CI).
