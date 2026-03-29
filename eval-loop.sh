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
VERBOSE=false
START_ROUND=""   # empty = auto-detect next round
PARALLEL=""      # empty = auto (parallel); "serial" = sequential subagents

usage() {
  cat <<'EOF'
Usage: ./eval-loop.sh [options]

Options:
  -n, --rounds <N>      Number of rounds (default: 4)
  -s, --start <N>       Starting round number (default: auto-detect next)
  -j, --jobs <N>        Subagent parallelism: N (default: parallel)
                       0 or "serial" — run subagents one at a time
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
  ./eval-loop.sh -s 10 -n 3          # run rounds 10, 11, 12
  ./eval-loop.sh -j 0                 # serial mode (reduce load)
  ./eval-loop.sh -d "focus on cross-language duplicate detection"
  ./eval-loop.sh -d "explore edge cases in task dependencies"
  ./eval-loop.sh --dry-run
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--rounds)   ROUNDS="$2";     shift 2 ;;
    -s|--start)    START_ROUND="$2"; shift 2 ;;
    -j|--jobs)     PARALLEL="$2";   shift 2 ;;
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

  # Subagent parallelism directive
  local parallel_directive=""
  if [[ "$PARALLEL" == "0" || "$PARALLEL" == "serial" ]]; then
    parallel_directive="
EXECUTION MODE: SERIAL
All subagents must be run SEQUENTIALLY (one at a time), NOT in parallel.
Launch one subagent, wait for it to complete, then launch the next.
This applies to BOTH Part 1 baseline and Part 2 verification phases."
  fi

  if [[ -n "$DESC" ]]; then
    cat <<EOF
You are running Round ${round}/${total} of the agent improvement loop.

Read this doc first:
- eval-loop.md

Execute ONE complete round following the v2 methodology.
${parallel_directive}
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
${parallel_directive}
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
echo "  Start round       : ${START_ROUND:-<auto-detect>}"
echo "  Subagent mode     : ${PARALLEL:-<parallel>}"
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

# --- Auto-detect start round ---
if [[ -z "$START_ROUND" ]]; then
  # Find the highest existing round number from log files, directories, and artifacts
  # Match patterns: round-{N}.log, round-{N}/, round-{N}-*.md, round-{N}-*.txt
  highest=0
  for f in "${SCRIPT_DIR}"/.eval/round-*.log \
           "${SCRIPT_DIR}"/.eval/round-[0-9]*.md \
           "${SCRIPT_DIR}"/.eval/round-[0-9]*.txt; do
    [[ -e "$f" ]] || continue
    basename=$(basename "$f")
    # Extract leading number from round-{N}... filenames (skip non-numeric like round-new-*)
    num=$(echo "$basename" | grep -oP '^round-\K\d+')
    [[ -n "$num" ]] && [[ "$num" -gt "$highest" ]] && highest=$num
  done
  for d in "${SCRIPT_DIR}"/.eval/round-[0-9]*/; do
    [[ -d "$d" ]] || continue
    basename=$(basename "$d")
    num=${basename#round-}
    num=${num%/}
    [[ "$num" =~ ^[0-9]+$ ]] && [[ "$num" -gt "$highest" ]] && highest=$num
  done
  START_ROUND=$((highest + 1))
fi

END_ROUND=$(( START_ROUND + ROUNDS - 1 ))

echo "  Round range       : ${START_ROUND} → ${END_ROUND}"
echo ""

START_TIME=$(date +%s)

# --- Loop ---
for i in $(seq "$START_ROUND" "$END_ROUND"); do
  echo "─────────────────────────────────────────────"
  echo "  ROUND ${i}/${END_ROUND}"
  echo "─────────────────────────────────────────────"

  if [[ "$DRY_RUN" == true ]]; then
    echo "  [DRY RUN] claude -p${MODEL_FLAG:+ $MODEL_FLAG}${VERBOSE_FLAG:+ $VERBOSE_FLAG}"
    echo ""
    echo "  ═════════════════════════════════════════"
    echo "  FULL PROMPT:"
    echo "  ═════════════════════════════════════════"
    echo ""
    build_prompt "$i" "$END_ROUND" | sed 's/^/  /'
    echo ""
    echo "  ═════════════════════════════════════════"
    echo ""
    continue
  fi

  ROUND_START=$(date +%s)

  build_prompt "$i" "$END_ROUND" | env -u CLAUDECODE claude -p $MODEL_FLAG $VERBOSE_FLAG 2>&1 | tee "${SCRIPT_DIR}/.eval/round-${i}.log"

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
echo "  Logs: ${SCRIPT_DIR}/.eval/round-{${START_ROUND}..${END_ROUND}}.log"
echo "═══════════════════════════════════════════"
