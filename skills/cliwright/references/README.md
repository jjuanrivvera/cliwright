# references

On-demand deep dives for the cliwright skill. `GOAL.md` (this skill's directory) is the canonical,
self-contained playbook; these files are extracts the agent can load when a specific
phase needs detail, without paying for the whole playbook up front.

- The **standard** (non-negotiable practices) → GOAL.md §1
- **Architecture** blueprint → GOAL.md §2
- **Meta-command set** → GOAL.md §3
- **MCP server + agent guard** → GOAL.md §3b
- **Beyond the API** (apply/lint/diff/backup/sync) → GOAL.md §3c
- **Distribution / CI/CD** → GOAL.md §5–§6
- **Determinism rules** → GOAL.md §11
- **Method enumeration + the two-way spec gate** (consistency via `spec-check`, completeness via
  `spec-completeness`) → GOAL.md §0 Step 1b + §11
- **Acceptance gate (`make verify`)** → GOAL.md §12

Extract individual files here only when a deep dive is large enough to warrant its own
progressive-disclosure load; otherwise keep the single source of truth in GOAL.md.
