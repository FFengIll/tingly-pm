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

# --- Banner ---
echo "╔══════════════════════════════════════════╗"
echo "║       eval-loop · Claude Code           ║"
echo "╚══════════════════════════════════════════╝"
echo ""
echo "  Working directory : ${SCRIPT_DIR}"
echo "  Rounds            : ${ROUNDS}"
echo "  Model             : ${MODEL:-<default>}"
echo "  Prompt file       : ${PROMPT_FILE:-<built-in>}"
echo "  Dry run           : ${DRY_RUN}"
echo "  Output dir        : ${SCRIPT_DIR}/.eval/"
echo ""

if [[ "$DRY_RUN" == true ]]; then
  echo "  *** DRY RUN — no commands will be executed ***"
  echo ""
fi

# Ensure output directory exists
if [[ "$DRY_RUN" == false ]]; then
  mkdir -p "${SCRIPT_DIR}/.eval"
fi

MODEL_FLAG=""
[[ -n "$MODEL" ]] && MODEL_FLAG="--model $MODEL"

START_TIME=$(date +%s)

# --- Loop ---
for i in $(seq 1 "$ROUNDS"); do
  echo "─────────────────────────────────────────────"
  echo "  ROUND ${i}/${ROUNDS}"
  echo "─────────────────────────────────────────────"

  if [[ "$DRY_RUN" == true ]]; then
    echo "  [DRY RUN] claude -p${MODEL_FLAG:+ $MODEL_FLAG}"
    echo "  [DRY RUN] prompt preview: $(build_prompt $i $ROUNDS | head -1)..."
    echo ""
    continue
  fi

  ROUND_START=$(date +%s)

  build_prompt "$i" "$ROUNDS" | claude -p $MODEL_FLAG 2>&1 | tee "${SCRIPT_DIR}/.eval/round-${i}.log"

  ROUND_END=$(date +%s)
  ROUND_ELAPSED=$(( ROUND_END - ROUND_START ))

  echo ""
  echo "  ✓ Round ${i} done (${ROUND_ELAPSED}s)"
  echo ""
done

END_TIME=$(date +%s)
TOTAL_ELAPSED=$(( END_TIME - START_TIME ))
TOTAL_MIN=$(( TOTAL_ELAPSED / 60 ))
TOTAL_SEC=$(( TOTAL_ELAPSED % 60 ))

echo "═══════════════════════════════════════════"
echo "  DONE. ${ROUNDS} round(s) completed in ${TOTAL_MIN}m ${TOTAL_SEC}s"
echo "  Logs: ${SCRIPT_DIR}/.eval/round-{1..${ROUNDS}}.log"
echo "═══════════════════════════════════════════"
