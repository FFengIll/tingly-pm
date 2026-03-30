# Test Fixtures

Stream JSON test fixtures for the eval loop. Each `.jsonl` file is a sequence of user messages (one per line). Pipe to `tingly-pm -mode run` via stdin.

## Execution

```bash
# Run a single fixture
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null

# Run automated assertions
./eval-assert.sh                    # all fixtures
./eval-assert.sh smoke              # smoke tests only
./eval-assert.sh create-task-english  # single fixture
./eval-assert.sh -v create-task-english  # verbose
```

Timeout by turn count: 1 turn=15s, 2-3 turns=30s, 4-5 turns=60s, 6+ turns=90s.

## Smoke Tests (Mandatory)

| Fixture | Category | Description |
|---------|----------|-------------|
| create-task-english | create | English input, explicit priority |
| create-task-chinese | create | Chinese input, keyword priority detection |
| update-task-single-field | update | No hallucination on update |
| create-task-duplicate | create | Dedup detection |
| error-empty-input | error | Empty content handling |

## Single-Turn (1 message)

| Fixture | Category | Description |
|---------|----------|-------------|
| create-task-english | create | English input, explicit priority |
| create-task-chinese | create | Chinese input, keyword priority detection |
| create-task-priority-keyword | create | Priority from Chinese keyword (紧急) |
| list-tasks-empty | list | Empty board listing |
| search-tasks-by-title | search | Full-text search by title keyword |
| member-register-list | member | Register member then list |
| member-labels-types | member | Register with labels & type filtering |
| member-error-scenarios | member | Duplicate registration & error handling |
| error-empty-input | error | Empty content handling |
| language-english-input | language | English input with priority filter |
| report-daily | report | Daily report generation |
| timeline-recent | timeline | Recent activity listing |
| summary-stats | summary | Quick project stats |
| mutated-member-typos | member | Member name typo (alic instead of alice) |
| mutated-empty-member-name | error | Empty member name registration |
| mutated-member-special-chars | member | Chinese character name (张三) |
| mutated-member-label-special-chars | member | Special characters in labels (前端@#$%) |
| mutated-member-missing-fields | error | Missing member name — tests validation |
| discovered-member-type-validation | error | Invalid member type — tests type validation |
| discovered-member-case-sensitivity | member | Case sensitivity: alice vs Alice |
| discovered-member-label-edge-cases | member | Empty/malformed labels |
| verify-special-chars-label | member | Special characters in member name |
| verify-overflow-title | create | Extremely long task title |

## Multi-Turn (2+ messages)

| Fixture | Turns | Category | Description |
|---------|-------|----------|-------------|
| create-task-duplicate | 2 | create | Create task then attempt duplicate → dedup detection |
| create-task-assign | 2 | create | Register member then create+assign task |
| update-task-single-field | 2 | update | Create task then update single field (priority) |
| mutated-assign-nonexistent-member | 2 | error | Create task then assign to non-existent member |
| mutated-assign-empty-members | 2 | error | Create task then assign to empty member list |
| context-resolve-by-name | 3 | context | Create task, update by name, list |
| context-cross-language | 3 | context | Chinese create, English update by reference |
| context-error-recovery | 3 | context | Create → invalid op → continue normally |
| workflow-create-comment-list | 3 | workflow | Create task → add comment → get detail |
| mutated-empty-comment | 3 | error | Empty comment content |
| mutated-cross-lang-error-injection | 3 | error | Error injection in cross-language workflow |
| mutated-cross-lang-update | 3 | context | Chinese create, mixed-language update |
| mutated-redundant-tool-usage | 3 | tool | Compound assign+status update request |
| mutated-reordered-comment-detail | 4 | workflow | Reordered comment+detail workflow |
| context-ordinal-reference | 4 | context | Create labeled tasks (A/B), resolve ordinal reference |
| workflow-create-assign-list | 4 | workflow | Create → register member → assign → list |
| mutated-cross-language-members | 5 | workflow | Mix of Chinese/English member registration |
| context-descriptive-reference | 5 | context | Create three tasks, resolve descriptive reference |
| workflow-dependency-add-remove | 5 | workflow | Create two → add dep → remove dep → list |
| discovered-tool-conflict-upsert | 5 | tool | UpsertMember register/update/register conflict |
| workflow-create-dep-archive | 5 | workflow | Full lifecycle: create → dep → archive → list |
| workflow-register-assign-multi | 6 | workflow | Register 2 members → create 2 tasks → assign each → list |
| mutated-redundant-list-members | 6 | tool | Alternating search/list member pattern |
| discovered-tool-ambiguity | 6 | tool | Multiple listing/reporting tools on empty board |
| member-labels-types | 7 | member | Multi-turn label filtering workflow |
| discovered-crud-consolidation | 7 | tool | Full CRUD lifecycle chaining |
| discovered-member-removal-workflow | 7 | workflow | Register → create tasks → remove member → verify |
| mutated-dep-add-remove-rapid | 8 | workflow | Rapid add/remove/re-add dependency cycle |
| discovered-cross-language-dedup | 3 | context | Cross-language duplicate detection |
| discovered-rapid-tool-chaining | 11 | tool | Rapid sequential tool invocation (3 members, 3 tasks) |
| discovered-complex-dependency-overload | 12 | tool | Complex 4-task dependency graph |
| mutated-label-overload | 12 | tool | Many labels per entity, filter queries |
| discovered-tool-redundancy-check | 11 | tool | Redundant list and duplicate assign operations |
| report-types-session | 4 | report | Weekly report + summary + session save |

## Fixture Lifecycle

- **Round 0**: Initial set (this file)
- **Round N**: Subagents MUTATE (copy + edit) or DISCOVER (new file) → update this INDEX
- Fixtures are git-tracked eval artifacts in `.eval/`
- Every 3 rounds: identify semantically duplicate fixtures, merge, update INDEX
- All multi-turn fixtures should have a companion `.expect.md` with per-turn grading criteria

## Automated Assertions

`eval-assert.sh` provides programmatic PASS/FAIL by parsing tool call patterns in agent output. It checks:
- Minimum response count (agent responds to every turn)
- Tool name presence (correct tools invoked)
- Output content patterns (task IDs, empty state messages)

Use `eval-assert.sh` as a reproducible regression gate — results are deterministic given the same binary and fixture.
