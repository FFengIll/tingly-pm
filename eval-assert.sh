#!/usr/bin/env bash
# eval-assert.sh — Batch-run fixtures, collect results, AI judges pass/fail
#
# Usage:
#   ./eval-assert.sh                        # run all fixtures, AI judge
#   ./eval-assert.sh smoke                  # smoke tests only
#   ./eval-assert.sh create-task-english    # single fixture
#   ./eval-assert.sh -v create-task-english # verbose (show raw output)
#   ./eval-assert.sh --collect-only         # run fixtures, skip AI judge
#   ./eval-assert.sh --judge-only           # re-judge existing results
#
# Flow:
#   1. Batch-execute all fixtures → collect raw output per fixture
#   2. Concatenate inputs + outputs + expect.md into one file
#   3. Send to AI for batch pass/fail judgment
#
# Exit code: 0 = all pass, 1 = any fail, 2 = usage error.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"
BINARY="${TINGLY_PM_BIN:-./tingly-pm}"
CONFIG_DIR="${TINGLY_PM_CONFIG:-.pm}"
EVAL_DIR="$SCRIPT_DIR/.eval"
VERBOSE="${VERBOSE:-0}"
COLLECT_ONLY=false
JUDGE_ONLY=false
BASE_TIMEOUT=15

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- Argument parsing ---

positional=()
while [[ $# -gt 0 ]]; do
  case $1 in
    -v|--verbose)        VERBOSE=1; shift ;;
    --collect-only)      COLLECT_ONLY=true; shift ;;
    --judge-only)        JUDGE_ONLY=true; shift ;;
    smoke)               positional+=("smoke"); shift ;;
    -h|--help)
      sed -n '2,/^$/s/^# //p' "$0"
      exit 0
      ;;
    -*) echo "Unknown option: $1"; exit 2 ;;
    *) positional+=("$1"); shift ;;
  esac
done

# --- Smoke test set ---

SMOKE_TESTS=(
  "create-task-english"
  "create-task-chinese"
  "update-task-single-field"
  "create-task-duplicate"
  "error-empty-input"
)

# --- Helpers ---

count_turns() {
  grep -c '' "$1" 2>/dev/null || echo 0
}

get_timeout() {
  local turns=$1
  if [ "$turns" -le 1 ]; then echo $BASE_TIMEOUT
  elif [ "$turns" -le 3 ]; then echo 30
  elif [ "$turns" -le 5 ]; then echo 60
  else echo 90
  fi
}

# --- Phase 1: Collect ---

collect() {
  local target="${1:-all}"

  # Resolve fixture list
  local fixtures=()
  if [[ "$target" == "all" ]]; then
    for f in "$FIXTURES_DIR"/*.jsonl; do
      [[ -f "$f" ]] && fixtures+=("$f")
    done
  elif [[ "$target" == "smoke" ]]; then
    for name in "${SMOKE_TESTS[@]}"; do
      local f="$FIXTURES_DIR/${name}.jsonl"
      [[ -f "$f" ]] && fixtures+=("$f")
    done
  else
    local f="$FIXTURES_DIR/${target}.jsonl"
    [[ ! -f "$f" ]] && { echo -e "${RED}Fixture not found: $target${NC}"; exit 2; }
    fixtures+=("$f")
  fi

  local batch_file="$EVAL_DIR/batch-results.md"
  mkdir -p "$EVAL_DIR"

  echo -e "${CYAN}Collecting results for ${#fixtures[@]} fixture(s)...${NC}"

  # Header
  {
    echo "# Eval Batch Results"
    echo ""
    echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Binary: $BINARY"
    echo "Fixtures: ${#fixtures[@]}"
    echo ""
    echo "---"
    echo ""
  } > "$batch_file"

  local pass=0 fail=0 skip=0 total=${#fixtures[@]}

  for fixture in "${fixtures[@]}"; do
    local name="$(basename "$fixture" .jsonl)"
    local turns="$(count_turns "$fixture")"
    local timeout="$(get_timeout "$turns")"
    local tmpdir="$(mktemp -d)"

    # Copy config
    mkdir -p "$tmpdir/.pm"
    if [[ -d "$CONFIG_DIR" ]]; then
      cp -r "$CONFIG_DIR"/* "$tmpdir/.pm/" 2>/dev/null || true
    fi

    local output="$tmpdir/output.jsonl"
    local config_arg=""
    [[ -d "$CONFIG_DIR" ]] && config_arg="-config $CONFIG_DIR"

    if [[ "$VERBOSE" == "1" ]]; then
      echo -e "  ${CYAN}Running $name (${turns} turns, ${timeout}s)${NC}"
    fi

    cat "$fixture" | timeout "$timeout" "$BINARY" -mode run -dir "$tmpdir" $config_arg 2>/dev/null > "$output" || true

    # Check basic sanity: got at least some output
    local response_count
    response_count="$(grep -c '"role":"assistant"' "$output" 2>/dev/null || echo 0)"

    {
      echo "## Fixture: $name"
      echo ""
      echo "### Input ($turns turns)"
      echo '```jsonl'
      cat "$fixture"
      echo '```'
      echo ""
      echo "### Output ($response_count responses)"
      echo '```jsonl'
      cat "$output"
      echo '```'
      echo ""
      # Include expect.md if it exists
      local expect="$FIXTURES_DIR/${name}.expect.md"
      if [[ -f "$expect" ]]; then
        echo "### Expectations"
        echo '```markdown'
        cat "$expect"
        echo '```'
        echo ""
      fi
      echo "---"
      echo ""
    } >> "$batch_file"

    # Basic sanity check
    if [[ "$response_count" -ge 1 ]]; then
      echo -e "  ${GREEN}OK${NC}: $name ($response_count responses)"
      ((pass++)) || true
    else
      echo -e "  ${RED}FAIL${NC}: $name (no responses)"
      ((fail++)) || true
    fi

    rm -rf "$tmpdir"
  done

  echo ""
  echo -e "  Collected: ${GREEN}$pass ok${NC}, ${RED}$fail fail${NC}, $total total"
  echo -e "  Batch file: $batch_file"
  echo ""

  # Return fail count as exit code for phase 1
  return "$fail"
}

# --- Phase 2: AI Judge ---

judge() {
  local batch_file="$EVAL_DIR/batch-results.md"
  local result_file="$EVAL_DIR/batch-verdict.md"

  if [[ ! -f "$batch_file" ]]; then
    echo -e "${RED}No batch results found. Run collection first.${NC}"
    exit 2
  fi

  echo -e "${CYAN}Sending batch to AI for judgment...${NC}"

  local judge_prompt
  judge_prompt=$(cat <<'JUDGE_EOF'
You are an eval judge for a project management agent (tingly-pm).

You will receive a batch of test fixture results. Each fixture has:
- **Input**: the user messages sent to the agent (JSONL)
- **Output**: the agent's raw responses (JSONL, each line is an assistant turn with tool_calls)
- **Expectations** (if present): what the fixture intends to test

For EACH fixture, determine PASS or FAIL based on these criteria:
1. The agent responded to every turn (response count >= turn count)
2. The agent called appropriate tools (correct tool names for the scenario)
3. The agent did not crash or produce errors on valid input
4. The agent's behavior matches the expectations if provided

For edge-case/error fixtures (empty input, missing fields, etc.): PASS if the agent handles it gracefully (no crash, helpful error message).

Output format — a markdown table followed by summary:

| # | Fixture | Verdict | Reason |
|---|---------|---------|--------|
| 1 | create-task-english | PASS | Called CreateTask, returned task ID |
| 2 | ... | ... | ... |

## Summary
- Total: N
- PASS: X
- FAIL: Y
- Pass rate: Z%

Be thorough but fair. The agent is non-deterministic — small variations in output are fine as long as the core behavior is correct.
JUDGE_EOF
)

  # Combine: judge prompt + batch results, send to Claude
  {
    echo "$judge_prompt"
    echo ""
    echo "# Batch Results"
    echo ""
    cat "$batch_file"
  } | claude -p --model sonnet 2>&1 | tee "$result_file"

  echo ""
  echo -e "  Verdict: $result_file"

  # Parse result for exit code
  if grep -qi 'FAIL' "$result_file" && ! grep -qi 'FAIL.*0' "$result_file"; then
    return 1
  fi
  return 0
}

# --- Main ---

main() {
  if [[ "$JUDGE_ONLY" == true ]]; then
    judge
    return $?
  fi

  collect "${positional[0]:-all}"

  if [[ "$COLLECT_ONLY" == true ]]; then
    return 0
  fi

  judge
}

# Banner
echo -e "${CYAN}tingly-pm eval (AI-judged)${NC}"

# Preflight
if [[ "$JUDGE_ONLY" == false ]]; then
  if [[ ! -x "$BINARY" ]]; then
    echo -e "${RED}Error: $BINARY not found or not executable${NC}"
    echo "Build first: go build -o tingly-pm ."
    exit 2
  fi
  if [[ ! -d "$FIXTURES_DIR" ]]; then
    echo -e "${RED}Error: fixtures directory not found: $FIXTURES_DIR${NC}"
    exit 2
  fi
fi

main
