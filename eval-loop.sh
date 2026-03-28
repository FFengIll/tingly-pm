#!/usr/bin/env bash
set -euo pipefail

# eval-loop.sh — Trigger N rounds of agent improvement via Claude Code
#
# Usage: ./eval-loop.sh [options]
#
# The script only controls the OUTER loop (how many rounds).
# Everything else (baseline, fuzzing, experiment, verify, commit/revert)
# is handled inside Claude Code via prompt.
#
# Output: .eval/round-{N}.log

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

ROUNDS=4
MODEL=""
PROMPT_FILE=""
DRY_RUN=false

usage() {
  cat <<'EOF'
Usage: ./eval-loop.sh [options]

Options:
  -n, --rounds <N>      Number of rounds (default: 4)
  -m, --model <model>   Claude model (omit for default)
  -p, --prompt <file>   Custom prompt file
      --dry-run         Print commands without executing
  -h, --help            Show help

Examples:
  ./eval-loop.sh
  ./eval-loop.sh -n 5 -m opus
  ./eval-loop.sh --dry-run
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--rounds)   ROUNDS="$2";     shift 2 ;;
    -m|--model)    MODEL="$2";      shift 2 ;;
    -p|--prompt)   PROMPT_FILE="$2"; shift 2 ;;
    --dry-run)     DRY_RUN=true;    shift ;;
    -h|--help)     usage ;;
    *) echo "Unknown option: $1"; usage ;;
  esac
done

# Build prompt
build_prompt() {
  local round=$1 total=$2
  if [[ -n "$PROMPT_FILE" && -f "$PROMPT_FILE" ]]; then
    cat "$PROMPT_FILE"
    return
  fi
  cat <<EOF
You are running Round ${round}/${total} of the agent improvement loop.

Read these docs first:
- .sdlc/docs/agent-improvement-methodology-20260328.spec.md
- .sdlc/docs/pm-improvement-playbook-20260328.spec.md
- .sdlc/docs/pm-improvement-log-20260328.spec.md

Execute ONE complete round following the v2 methodology.
Improve the agent, verify improvements, then commit or revert.
Append results to the improvement log.

Output directory: .eval/
Write per-round artifacts there (baseline results, experiment reports, etc).
EOF
}

MODEL_FLAG=""
[[ -n "$MODEL" ]] && MODEL_FLAG="--model $MODEL"

# Loop
for i in $(seq 1 "$ROUNDS"); do
  echo ""
  echo "══ ROUND ${i}/${ROUNDS} ══"

  if [[ "$DRY_RUN" == true ]]; then
    echo "[DRY RUN] claude -p${MODEL_FLAG:+ $MODEL_FLAG}"
    echo "[DRY RUN] prompt: $(build_prompt $i $ROUNDS | head -3)..."
    continue
  fi

  build_prompt "$i" "$ROUNDS" | claude -p $MODEL_FLAG 2>&1 | tee .eval/round-${i}.log

  echo "── ROUND ${i} done ──"
done

echo ""
echo "DONE. ${ROUNDS} rounds completed."
