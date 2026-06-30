---
description: "Forge a production-grade CLI for an API and loop until the acceptance gate passes"
argument-hint: "<API name> <docs or OpenAPI URL> [go module path]"
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch"]
---

# /cliwright:build-cli

Build a complete, agent-ready CLI for **$ARGUMENTS** following the cliwright playbook.

## Steps

1. Read the canonical playbook at `${CLAUDE_PLUGIN_ROOT}/skills/cliwright/GOAL.md`. It is self-contained —
   do not rely on reading any other repository.
2. Parse `$ARGUMENTS` into the playbook's **TARGET API** block:
   - `$1` → `api_name`
   - `$2` → `docs_url` (fetch it / the OpenAPI / llms.txt and auto-resolve auth, base URL,
     pagination, rate-limit headers, resources, fields, special ops, JSON quirks)
   - `$3` → `module_path` (optional; otherwise infer `github.com/<owner>/<api>-cli`)
3. Ask the user **only** what the web cannot know (their instance/host, scope, distribution
   identity), batched into ≤4 questions. Skip anything already supplied.
4. Execute the build phases in order, committing per phase.
5. **Loop to the gate.** Run under the native `/goal` loop and bind its completion promise
   to the acceptance gate: emit the promise only when `make verify` exits `0`
   (`make check` + coverage ≥80% + `spec-check` + the atomic Definition-of-Done checklist
   + the subjective-item judge rubric). Never emit a false promise to escape the loop.

When the gate is green, summarize: the command surface, how to `make build` / install, and
the next steps (live-test against a real instance, tag a release, package as a plugin/skill).
