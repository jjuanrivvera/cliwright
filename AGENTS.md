# AGENTS.md

Guidance for AI agents working **on cliwright itself** (not on a CLI it generates).

## What this is

`cliwright` is a **spec-gated CLI factory**: a Claude Code plugin + cross-tool agent skill
that generates production-grade CLIs for any API and loops (via the native `/goal`) until a
deterministic acceptance gate passes. The product is mostly **specification + packaging**,
not application code.

## Layout

```
GOAL.md                         the canonical, SELF-CONTAINED playbook (source of truth)
.claude-plugin/
  plugin.json                   Claude Code plugin manifest
  marketplace.json              self-distribution manifest
skills/cliwright/
  SKILL.md                      lean skill wrapper → points to GOAL.md
  references/                   deep-dives extracted from GOAL.md (loaded on demand)
  templates/                    copy-in skeletons (generic core, Makefile+verify, goreleaser, ci)
  scripts/                      ralph.sh (fallback loop), verify helpers
commands/build-cli.md           /cliwright:build-cli slash command
agents/cli-builder.md           subagent that runs the long build
```

## Rules when editing

- **GOAL.md is the source of truth.** SKILL.md, the command, and the agent all point to it;
  keep them thin. Put substantive changes in GOAL.md, not duplicated across wrappers.
- **GOAL.md must stay standalone.** Never make it depend on reading another repository
  (e.g. alegra-cli / canvas-cli). Inline every pattern; the §8 skeletons are the source.
- **Preserve the battle-tested specifics** (exact gosec/cosign/goreleaser invocations) —
  they encode real failure modes. Don't "simplify" them away.
- **The gate is the contract.** Any new quality bar must land as a checkable item in the
  acceptance gate (`make verify`), not as prose the agent can wave through.
- Comments/docs explain WHY. Keep edits consistent with the surrounding style.
