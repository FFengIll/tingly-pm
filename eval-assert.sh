#!/usr/bin/env bash
# eval-assert.sh — Automated fixture assertion runner for tingly-pm
#
# Usage:
#   ./eval-assert.sh                           # run all fixtures
#   ./eval-assert.sh smoke                     # smoke tests only
#   ./eval-assert.sh create-task-english        # single fixture
#   ./eval-assert.sh -v create-task-english     # verbose (show output)
#
# Each fixture is run against a fresh tingly-pm instance. Output is parsed
# for tool call patterns. Results are compared against expect.md assertions.
# Exit code: 0 = all pass, 1 = any fail, 2 = usage error.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"
BINARY="${TINGLY_PM_BIN:-./tingly-pm}"
CONFIG_DIR="${TINGLY_PM_CONFIG:-.pm}"
VERBOSE="${VERBOSE:-0}"
BASE_TIMEOUT=15

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass_count=0
fail_count=0
skip_count=0
total_count=0

log_pass() { echo -e "  ${GREEN}PASS${NC}: $1"; ((pass_count++)) || true; }
log_fail() { echo -e "  ${RED}FAIL${NC}: $1"; ((fail_count++)) || true; }
log_skip() { echo -e "  ${YELLOW}SKIP${NC}: $1"; ((skip_count++)) || true; }

# Count turns in a jsonl file
count_turns() {
  wc -l < "$1" | tr -d ' '
}

# Compute timeout based on turn count
get_timeout() {
  local turns=$1
  if [ "$turns" -le 1 ]; then echo $BASE_TIMEOUT
  elif [ "$turns" -le 3 ]; then echo 30
  elif [ "$turns" -le 5 ]; then echo 60
  else echo 90
  fi
}

# Run a single fixture and return the output file path
run_fixture() {
  local fixture="$1"
  local name="$(basename "$fixture" .jsonl)"
  local turns="$(count_turns "$fixture")"
  local timeout="$(get_timeout "$turns")"
  local tmpdir="$(mktemp -d)"

  # Copy config for session/timeline support
  cp -r "$CONFIG_DIR"/* "$tmpdir/" 2>/dev/null || true

  local output="$tmpdir/output.jsonl"

  if [ "$VERBOSE" = "1" ]; then
    echo -e "  ${CYAN}Running $name (${turns} turns, ${timeout}s timeout)${NC}"
  fi

  cat "$fixture" | timeout "$timeout" "$BINARY" -mode run -dir "$tmpdir" 2>/dev/null > "$output" || true
  echo "$output"
}

# --- Assertion Functions ---
# These parse the agent's JSON output and check structural properties.
# The output is line-delimited JSON: each line is {"role":"assistant","content":"...","tool_calls":[...]}

# Check that the output has at least N response lines (one per turn)
assert_min_responses() {
  local output="$1"
  local expected="$2"
  local actual="$(grep -c '"role":"assistant"' "$output" 2>/dev/null || echo 0)"
  if [ "$actual" -ge "$expected" ]; then
    log_pass "response count: $actual >= $expected"
    return 0
  else
    log_fail "response count: $actual < $expected"
    return 1
  fi
}

# Check that a tool call with a given name pattern appears in the output
assert_tool_called() {
  local output="$1"
  local tool_pattern="$2"  # grep pattern, e.g. "CreateTask" or "UpsertMember"
  local description="${3:-tool $tool_pattern called}"
  if grep -q "\"name\":\"$tool_pattern\"" "$output" 2>/dev/null; then
    log_pass "$description"
    return 0
  else
    log_fail "$description (not found in output)"
    if [ "$VERBOSE" = "1" ]; then
      echo -e "    ${CYAN}Output:${NC}"
      cat "$output" | head -5
    fi
    return 1
  fi
}

# Check that output does NOT contain a tool call pattern
assert_tool_not_called() {
  local output="$1"
  local tool_pattern="$2"
  local description="${3:-tool $tool_pattern NOT called}"
  if ! grep -q "\"name\":\"$tool_pattern\"" "$output" 2>/dev/null; then
    log_pass "$description"
    return 0
  else
    log_fail "$description (found unexpectedly)"
    return 1
  fi
}

# Check that output contains a text substring
assert_output_contains() {
  local output="$1"
  local pattern="$2"
  local description="${3:-output contains '$pattern'}"
  if grep -q "$pattern" "$output" 2>/dev/null; then
    log_pass "$description"
    return 0
  else
    log_fail "$description"
    return 1
  fi
}

# --- Smoke Test Suite ---

SMOKE_TESTS=(
  "create-task-english"
  "create-task-chinese"
  "update-task-single-field"
  "create-task-duplicate"
  "error-empty-input"
)

run_smoke() {
  echo -e "\n${CYAN}=== Smoke Tests ===${NC}"
  for name in "${SMOKE_TESTS[@]}"; do
    local fixture="$FIXTURES_DIR/${name}.jsonl"
    if [ ! -f "$fixture" ]; then
      log_skip "$name (fixture not found)"
      continue
    fi
    run_single "$fixture"
  done
}

# --- Per-fixture assertion definitions ---

run_single() {
  local fixture="$1"
  local name="$(basename "$fixture" .jsonl)"
  local turns="$(count_turns "$fixture")"
  ((total_count++)) || true

  echo -e "\n${CYAN}--- $name (${turns} turns) ---${NC}"

  local output
  output="$(run_fixture "$fixture")"

  # Check that we got responses for all turns
  assert_min_responses "$output" "$turns"

  # Run fixture-specific assertions
  case "$name" in
    create-task-english)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_output_contains "$output" "TASK-" "output contains task ID"
      ;;
    create-task-chinese)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_output_contains "$output" "TASK-" "output contains task ID"
      ;;
    create-task-priority-keyword)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_output_contains "$output" "TASK-" "output contains task ID"
      ;;
    create-task-duplicate)
      assert_tool_called "$output" "CreateTask" "CreateTask called (first time)"
      # Second turn should NOT call CreateTask again (dedup)
      ;;
    create-task-assign)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called for alice"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      ;;
    update-task-single-field)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    list-tasks-empty)
      assert_tool_called "$output" "ListTasks" "ListTasks called"
      assert_output_contains "$output" "No tasks" "empty board handled"
      ;;
    search-tasks-by-title)
      assert_tool_called "$output" "SearchTasks\|ListTasks" "search/list tool called"
      ;;
    member-register-list)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    member-labels-types)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    member-error-scenarios)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    error-empty-input)
      assert_output_contains "$output" "" "no crash on empty input" && true  # just check no crash
      ;;
    language-english-input)
      assert_tool_called "$output" "ListTasks\|SearchTasks" "task listing tool called"
      ;;
    report-daily)
      assert_tool_called "$output" "GenerateReport" "GenerateReport called"
      ;;
    timeline-recent)
      assert_tool_called "$output" "GenerateReport\|ListTimeline" "report/timeline tool called"
      ;;
    summary-stats)
      assert_tool_called "$output" "GenerateReport\|Summary" "report/summary tool called"
      ;;
    report-types-session)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "SaveSession" "SaveSession called"
      assert_tool_called "$output" "GenerateReport" "GenerateReport called"
      ;;
    verify-special-chars-label)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    verify-overflow-title)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      ;;
    mutated-member-typos)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    mutated-empty-member-name)
      # Should not crash — may or may not call tool
      assert_min_responses "$output" 1
      ;;
    mutated-member-special-chars)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    mutated-member-label-special-chars)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    mutated-label-overload)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    mutated-member-missing-fields)
      # Edge case: missing member name — should not crash
      assert_min_responses "$output" 1
      ;;
    discovered-member-type-validation)
      # Invalid member type — should not crash
      assert_min_responses "$output" 1
      ;;
    discovered-member-case-sensitivity)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    discovered-member-label-edge-cases)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      ;;
    context-resolve-by-name)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    context-ordinal-reference)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    context-descriptive-reference)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    context-cross-language)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    context-error-recovery)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_min_responses "$output" 3
      ;;
    mutated-cross-lang-update)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    mutated-cross-lang-error-injection)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_min_responses "$output" 3
      ;;
    mutated-cross-language-members)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    workflow-create-dep-archive)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "AddDependency" "AddDependency called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    workflow-create-assign-list)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      assert_tool_called "$output" "ListTasks" "ListTasks called"
      ;;
    workflow-dependency-add-remove)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "AddDependency" "AddDependency called"
      assert_tool_called "$output" "RemoveDependency" "RemoveDependency called"
      ;;
    workflow-create-comment-list)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask\|AddComment" "comment/update tool called"
      ;;
    workflow-register-assign-multi)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    mutated-empty-comment)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      ;;
    mutated-assign-nonexistent-member)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_min_responses "$output" 2
      ;;
    mutated-assign-empty-members)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_min_responses "$output" 2
      ;;
    mutated-dep-add-remove-rapid)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "AddDependency" "AddDependency called"
      assert_tool_called "$output" "RemoveDependency" "RemoveDependency called"
      ;;
    mutated-redundant-tool-usage)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    mutated-redundant-list-members)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    mutated-reordered-comment-detail)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask\|AddComment" "comment/update tool called"
      ;;
    discovered-tool-ambiguity)
      assert_tool_called "$output" "ListTasks\|SearchTasks" "task tool called"
      assert_tool_called "$output" "GenerateReport\|ListTimeline\|Summary" "report tool called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    discovered-tool-conflict-upsert)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    discovered-tool-redundancy-check)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "ListTasks" "ListTasks called"
      ;;
    discovered-rapid-tool-chaining)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      assert_tool_called "$output" "ListTasks" "ListTasks called"
      ;;
    discovered-complex-dependency-overload)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "AddDependency" "AddDependency called"
      assert_tool_called "$output" "UpdateTask" "UpdateTask called"
      ;;
    discovered-crud-consolidation)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      ;;
    discovered-cross-language-dedup)
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "ListTasks\|SearchTasks" "list/search tool called"
      ;;
    discovered-member-removal-workflow)
      assert_tool_called "$output" "UpsertMember" "UpsertMember called"
      assert_tool_called "$output" "CreateTask" "CreateTask called"
      assert_tool_called "$output" "RemoveMember" "RemoveMember called"
      assert_tool_called "$output" "ListMembers" "ListMembers called"
      assert_tool_called "$output" "ListTasks" "ListTasks called"
      ;;
    *)
      log_skip "no automated assertions for $name"
      ;;
  esac

  # Cleanup
  rm -rf "$(dirname "$output")"
}

# --- Main ---

main() {
  if [ ! -x "$BINARY" ]; then
    echo -e "${RED}Error: $BINARY not found or not executable${NC}"
    echo "Build first: go build -o tingly-pm ."
    exit 2
  fi

  if [ ! -d "$FIXTURES_DIR" ]; then
    echo -e "${RED}Error: fixtures directory not found: $FIXTURES_DIR${NC}"
    exit 2
  fi

  echo -e "${CYAN}tingly-pm eval assertions${NC}"
  echo -e "Binary: $BINARY"
  echo -e "Fixtures: $FIXTURES_DIR"
  echo -e "Config: $CONFIG_DIR"
  echo -e "Time: $(date '+%Y-%m-%d %H:%M:%S')"

  if [ $# -eq 0 ]; then
    # Run all fixtures
    for fixture in "$FIXTURES_DIR"/*.jsonl; do
      run_single "$fixture"
    done
  elif [ "$1" = "smoke" ]; then
    run_smoke
  else
    # Run specific fixture(s)
    for arg in "$@"; do
      local fixture="$FIXTURES_DIR/${arg}.jsonl"
      if [ ! -f "$fixture" ]; then
        fixture="$FIXTURES_DIR/${arg}"
      fi
      if [ ! -f "$fixture" ]; then
        echo -e "${RED}Error: fixture not found: $arg${NC}"
        exit 2
      fi
      run_single "$fixture"
    done
  fi

  # Summary
  echo -e "\n${CYAN}=== Summary ===${NC}"
  echo -e "  Total: $total_count fixtures"
  echo -e "  ${GREEN}Pass:  $pass_count${NC}"
  echo -e "  ${RED}Fail:  $fail_count${NC}"
  echo -e "  ${YELLOW}Skip:  $skip_count${NC}"

  if [ "$fail_count" -gt 0 ]; then
    exit 1
  fi
  exit 0
}

main "$@"
