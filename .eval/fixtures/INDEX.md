# Test Fixtures

Stream JSON test fixtures for the eval loop. Each `.jsonl` file is a sequence of user messages (one per line). Pipe to `tingly-pm -mode run` via stdin.

## Execution

```bash
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null
```

Timeout by turn count: 1 turn=30s, 2-3 turns=60s, 4-5 turns=120s, 6+ turns=180s.

## Single-Turn (1 message)

| Fixture | Category | Description |
|---------|----------|-------------|
| create-task-chinese.jsonl | create | Chinese input, keyword priority detection |
| create-task-english.jsonl | create | English input, explicit priority |
| create-task-priority-keyword.jsonl | create | Priority from Chinese keyword (紧急) |
| update-task-nonexistent.jsonl | error | Graceful error for nonexistent task ID |
| list-tasks-empty.jsonl | list | Empty board listing |
| search-tasks-by-title.jsonl | search | Full-text search by title keyword |
| member-register-list.jsonl | member | Register member then list |
| member-labels-types.jsonl | member | DISCOVERED: Register with labels & type filtering |
| member-error-scenarios.jsonl | error | DISCOVERED: Duplicate registration & non-existent assignee |
| error-empty-input.jsonl | error | Empty content handling |
| error-invalid-taskid.jsonl | error | Get nonexistent task by ID |
| language-english-input.jsonl | language | English input with priority filter |
| report-daily.jsonl | report | Daily report generation |
| timeline-recent.jsonl | timeline | Recent activity listing |
| summary-stats.jsonl | summary | Quick project stats |
| mutated-member-typos.jsonl | member | MUTATED FROM: member-register-list → Member name typo (alic instead of alice) |
| mutated-empty-member-name.jsonl | error | MUTATED FROM: member-register-list → Empty member name registration |
| mutated-member-special-chars.jsonl | member | MUTATED FROM: member-register-list → Chinese character name (张三) |
| mutated-member-label-special-chars.jsonl | member | MUTATED FROM: member-labels-types → Special characters in labels (前端@#$％) |
| mutated-label-overload.jsonl | 12 | member | MUTATED FROM: member-labels-types → Tests agent with excessive labels (6+ per entity) and complex filtering scenarios |
| discovered-conflicting-member-names.jsonl | error | DISCOVERED: Duplicate member registration — rationale: Tests duplicate detection |
| discovered-member-missing-fields.jsonl | error | DISCOVERED: Missing member name — rationale: Tests validation of required fields |
| discovered-member-label-edge-cases.jsonl | member | DISCOVERED: Empty/malformed labels — rationale: Tests label handling edge cases |
| discovered-member-type-validation.jsonl | error | DISCOVERED: Invalid member type — rationale: Tests type validation |
| discovered-member-case-sensitivity.jsonl | member | DISCOVERED: Case sensitivity in names — rationale: Tests name matching policy |
| discovered-tool-ambiguity.jsonl | 6 | tool | DISCOVERED BY SUBAGENT 1: Multiple empty-state listing tools — rationale: Tests tool redundancy when no data exists (search, list, report, timeline, summary, members) |
| discovered-tool-conflict-upsert.jsonl | 5 | tool | DISCOVERED BY SUBAGENT 2: UpsertMember conflicts — rationale: Tests RegisterMember vs UpsertMember vs UpdateMember tool confusion |

## Multi-Turn (2+ messages)

| Fixture | Turns | Category | Description |
|---------|-------|----------|-------------|
| create-task-duplicate.jsonl | 2 | create | Create task then attempt duplicate → dedup detection |
| create-task-assign.jsonl | 2 | create | Register member then create+assign task |
| update-task-single-field.jsonl | 2 | update | Create task then update single field (priority) |
| context-resolve-by-name.jsonl | 3 | context | Create task, update by name, list |
| context-implicit-reference.jsonl | 4 | context | Create two tasks, "第一个任务" reference, list |
| context-ordinal-reference.jsonl | 4 | context | EXP 4: Create labeled tasks (A/B), resolve "第一个任务" (ordinal) |
| context-descriptive-reference.jsonl | 5 | context | EXP 4: Create three tasks, resolve "那个登录任务" (descriptive) |
| workflow-create-dep-archive.jsonl | 5 | workflow | Full lifecycle: create → dep → archive → list |
| workflow-create-assign-list.jsonl | 4 | workflow | Create → register member → assign → list |
| context-cross-language.jsonl | 3 | context | Chinese create, English update by reference |
| context-error-recovery.jsonl | 3 | context | Create → invalid op → continue normally |
| workflow-dependency-add-remove.jsonl | 5 | workflow | Create two → add dep → remove dep → list |
| workflow-create-comment-list.jsonl | 3 | workflow | Create task → add comment → get detail |
| workflow-register-assign-multi.jsonl | 6 | workflow | Register 2 members → create 2 tasks → assign each → list |
| mutated-cross-language-members.jsonl | 4 | workflow | MUTATED FROM: workflow-create-assign-list → Cross-language member names |
| mutated-cross-lang-error-injection.jsonl | 3 | error | MUTATED FROM: context-cross-language → Error injection in cross-language workflow |
| mutated-empty-comment.jsonl | 3 | error | MUTATED FROM: workflow-create-comment-list → Empty comment content |
| mutated-assign-nonexistent-member.jsonl | 2 | error | MUTATED FROM: create-task-assign → Create task then assign to non-existent member |
| mutated-assign-empty-members.jsonl | 2 | error | MUTATED FROM: create-task-assign → Create task then assign to empty member list |
| mutated-redundant-tool-usage.jsonl | 3 | tool | MUTATED BY SUBAGENT 2: Register member, create+assign+update task — rationale: Tests if agent avoids redundant tool calls |
| mutated-dep-add-remove-rapid.jsonl | 8 | workflow | MUTATED: Register members, create tasks, add/remove deps rapidly |
| discovered-rapid-tool-chaining.jsonl | 11 | tool | DISCOVERED: Rapid sequential tool invocation — rationale: Tests agent's ability to handle many tools in quick succession (member creates, task creates, assignments, lists) |
| discovered-complex-dependency-overload.jsonl | 12 | tool | DISCOVERED: Complex dependency management — rationale: Tests multiple dependency additions and state transitions on interrelated tasks |
| discovered-tool-redundancy-check.jsonl | 11 | tool | DISCOVERED BY SUBAGENT 3: Tool redundancy & repeated operations — rationale: Tests whether agent avoids redundant tool calls for same operation (repeated lists, duplicate assignments) |

## Fixture Lifecycle

- **Round 0**: Initial set (this file)
- **Round N**: Subagents MUTATE (copy + edit) or DISCOVER (new file) → update this INDEX
- Fixtures are git-tracked eval artifacts in `.eval/`
