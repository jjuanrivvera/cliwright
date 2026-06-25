<div align="center">

# cliwright

**A wright of CLIs.** Forge a complete, production-grade, agent-ready command-line tool for *any* REST API — and loop until it provably meets every quality bar.

</div>

`cliwright` is a **spec-gated CLI factory**. Point it at an API; it researches the docs,
scaffolds a Go + Cobra + GoReleaser binary to a single high standard (keyring auth, named
profiles, resilient client, an MCP server, an `agent guard`, CI/CD, signed releases,
packaging), and **drives the build to a deterministic acceptance gate** so "done" means
*provably done*, not *asserted done*.

It is not a new agent loop — it rides the runtime's native **`/goal`** loop (Claude Code
and Codex both ship it) and supplies the two things a loop needs to finish honestly: a
complete **spec** and a machine-checkable **gate** the completion promise is bound to.

## What you get

- **One standard, every API.** A generic typed core + thin per-resource files; full CRUD,
  pagination, idempotent retry, adaptive rate limiting, `--dry-run` curl, Ctrl-C cancel.
- **Secure by construction.** Tokens in the OS keyring, secret flags never exposed to the
  MCP surface, path-confined file reads, CSV formula-injection guards.
- **Agent-ready.** Every command annotated read-only/write/destructive; an `mcp` server and
  an `agent guard` derived from the live command tree.
- **Shipped completely.** Homebrew/Scoop + deb/rpm/apk, cosign + SBOM, generated docs site,
  and — optionally — its own Claude Code plugin + cross-tool agent skill.

The full, self-contained playbook is **[GOAL.md](GOAL.md)**.

## Install

**As a Claude Code plugin**

```text
/plugin marketplace add jjuanrivvera/cliwright
/plugin install cliwright@cliwright
```

**As a cross-tool agent skill** (Cursor, Codex, Gemini CLI, …)

```bash
npx skills add jjuanrivvera/cliwright
```

## Use

```text
# Slash command (Claude Code)
/cliwright:build-cli Stripe https://docs.stripe.com/api github.com/me/stripe-cli

# Or paste GOAL.md after /goal, fill the TARGET API block, and let the loop run.
```

The build is complete when the acceptance gate is green:

```bash
make verify   # make check + coverage + spec-check + Definition-of-Done checklist + judge rubric
```

## How it works

| Layer | Responsibility |
|---|---|
| **`/goal`** (native) | The loop engine — re-feeds the goal until completion |
| **`cliwright` skill / `GOAL.md`** | The spec: standard, architecture, phases, determinism rules |
| **`make verify`** | The deterministic gate the completion promise binds to |
| **`cli-builder` agent** | Runs the long multi-phase build in its own context window |

## Development

The plugin manifests (`.claude-plugin/plugin.json`, `.claude-plugin/marketplace.json`)
are validated against the vendored schemastore schemas in `.claude-plugin/schemas/`,
plus structural checks that every referenced component resolves and carries the frontmatter
Claude Code needs to load it.

```bash
python3 scripts/validate-plugin.py   # run the same check CI runs
./scripts/install-hooks.sh           # enable the pre-commit hook (sets core.hooksPath)
```

CI runs the validator on every push and PR (`.github/workflows/validate.yml`). The full
schema layer needs `jsonschema` (`pip install jsonschema`); without it the hook still runs
the structural checks and CI enforces the rest. To refresh the vendored schemas, re-download
[`claude-code-plugin-manifest.json`](https://json.schemastore.org/claude-code-plugin-manifest.json)
and [`claude-code-marketplace.json`](https://json.schemastore.org/claude-code-marketplace.json)
into `.claude-plugin/schemas/`.

## License

MIT — see [LICENSE](LICENSE).
