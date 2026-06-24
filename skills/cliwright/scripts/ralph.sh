#!/usr/bin/env bash
# ralph.sh — portable fallback loop for cliwright.
#
# Use this ONLY when the runtime has no native `/goal` loop. It re-feeds the build
# prompt to a headless agent until the acceptance gate (`make verify`) exits 0.
# The gate is the source of truth, so the loop cannot "lie" its way to done.
#
# Usage:
#   CLIWRIGHT_AGENT=claude ./ralph.sh                 # default: claude -p, 30 iters
#   CLIWRIGHT_AGENT=codex  ./ralph.sh --max 20
#   ./ralph.sh --goal GOAL.md --prompt "Continue the build."
set -euo pipefail

AGENT="${CLIWRIGHT_AGENT:-claude}"
MAX="${CLIWRIGHT_MAX_ITERS:-30}"
GOAL="${CLIWRIGHT_GOAL:-GOAL.md}"
PROMPT_OVERRIDE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent)  AGENT="$2"; shift 2 ;;
    --max)    MAX="$2"; shift 2 ;;
    --goal)   GOAL="$2"; shift 2 ;;
    --prompt) PROMPT_OVERRIDE="$2"; shift 2 ;;
    -h|--help) sed -n '2,14p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

PROMPT="${PROMPT_OVERRIDE:-Continue building this CLI strictly per ${GOAL}. Run \`make verify\`; if it fails, read the failure, fix the smallest thing, and continue. Do NOT stop or claim completion until \`make verify\` exits 0.}"

run_agent() {
  case "$AGENT" in
    claude) claude -p "$PROMPT" ;;
    codex)  codex exec "$PROMPT" ;;
    *) echo "✗ unknown agent '$AGENT' (want: claude | codex)" >&2; exit 2 ;;
  esac
}

i=0
while (( i < MAX )); do
  if make verify; then
    echo "✓ acceptance gate passed after ${i} agent iteration(s)"
    exit 0
  fi
  i=$(( i + 1 ))
  echo "── ralph iteration ${i}/${MAX} (agent: ${AGENT}) ──"
  run_agent || echo "  (agent invocation returned non-zero; re-checking the gate anyway)"
done

echo "✗ acceptance gate still failing after ${MAX} iterations — stopping for human review" >&2
exit 1
