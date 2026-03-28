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
DESC=""
DRY_RUN=false
VERBOSE=true

usage() {
  cat <<'EOF'
Usage: ./eval-loop.sh [options]

Options:
  -n, --rounds <N>      Number of rounds (default: 4)
  -m, --model <model>   Claude model (omit for default)
  -p, --prompt <file>   Custom prompt file
  -d, --desc <text>     Additional description (fuzzing seed/mutator)
                       Used to guide exploration focus or add constraints
  -v, --verbose         Show Claude's intermediate steps (default: on)
      --no-verbose       Hide intermediate steps, output only final result
      --dry-run         Print commands without executing
  -h, --help            Show help

Examples:
  ./eval-loop.sh
  ./eval-loop.sh -n 5 -m opus
  ./eval-loop.sh -d "focus on cross-language duplicate detection"
  ./eval-loop.sh -d "explore edge cases in task dependencies"
  ./eval-loop.sh --dry-run
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--rounds)   ROUNDS="$2";     shift 2 ;;
    -m|--model)    MODEL="$2";      shift 2 ;;
    -p|--prompt)   PROMPT_FILE="$2"; shift 2 ;;
    -d|--desc)     DESC="$2";       shift 2 ;;
    --dry-run)     DRY_RUN=true;    shift ;;
    --no-verbose)  VERBOSE=false;   shift ;;
    -v|--verbose)   VERBOSE=true;    shift ;;
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

  if [[ -n "$DESC" ]]; then
    cat <<EOF
You are running Round ${round}/${total} of the agent improvement loop.

Read this doc first:
- eval-loop.md

Execute ONE complete round following the v2 methodology.

EXPLORATION SEED: ${DESC}

Use this seed to guide your improvement strategy:
- If it specifies a focus area, prioritize tests and experiments in that area
- If it suggests constraints, ensure all experiments respect them
- If it proposes a direction, favor hypotheses aligned with it
- Document in the final report how this seed influenced the round

Improve the agent, verify improvements, then commit or revert.
Append results to the improvement log.

Output directory: .eval/
Write per-round artifacts there (baseline results, experiment reports, etc).
EOF
  else
    cat <<EOF
You are running Round ${round}/${total} of the agent improvement loop.

Read this doc first:
- eval-loop.md

Execute ONE complete round following the v2 methodology.
Improve the agent, verify improvements, then commit or revert.
Append results to the improvement log.

Output directory: .eval/
Write per-round artifacts there (baseline results, experiment reports, etc).
EOF
  fi
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
echo "  Exploration seed  : ${DESC:-<none>}"
echo "  Dry run           : ${DRY_RUN}"
echo "  Verbose           : ${VERBOSE}"
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
VERBOSE_FLAG=""
[[ "$VERBOSE" == true ]] && VERBOSE_FLAG="--verbose"

START_TIME=$(date +%s)

# --- Loop ---
for i in $(seq 1 "$ROUNDS"); do
  echo "─────────────────────────────────────────────"
  echo "  ROUND ${i}/${ROUNDS}"
  echo "─────────────────────────────────────────────"

  if [[ "$DRY_RUN" == true ]]; then
    echo "  [DRY RUN] claude -p${MODEL_FLAG:+ $MODEL_FLAG}${VERBOSE_FLAG:+ $VERBOSE_FLAG}"
    echo ""
    echo "  ═════════════════════════════════════════"
    echo "  FULL PROMPT:"
    echo "  ═════════════════════════════════════════"
    echo ""
    build_prompt "$i" "$ROUNDS" | sed 's/^/  /'
    echo ""
    echo "  ═════════════════════════════════════════"
    echo ""
    continue
  fi

  ROUND_START=$(date +%s)

  build_prompt "$i" "$ROUNDS" | claude -p $MODEL_FLAG $VERBOSE_FLAG 2>&1 | tee "${SCRIPT_DIR}/.eval/round-${i}.log"

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
