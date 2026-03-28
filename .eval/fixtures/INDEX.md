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
| create-task-duplicate.jsonl | create | Duplicate detection (requires prior create in session) |
| create-task-priority-keyword.jsonl | create | Priority from Chinese keyword (紧急) |
| create-task-assign.jsonl | create | Create with member assignment |
| update-task-single-field.jsonl | update | Update single field (priority) |
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
| mutated-assign-nonexistent-member.jsonl | error | MUTATED FROM: create-task-assign → Assign to non-existent member |
| mutated-assign-empty-members.jsonl | error | MUTATED FROM: create-task-assign → Assign to empty member list |
| mutated-cross-language-members.jsonl | workflow | MUTATED FROM: workflow-create-assign-list → Cross-language member names |
| mutated-member-typos.jsonl | member | MUTATED FROM: member-register-list → Member name typo (alic instead of alice) |
| mutated-empty-member-name.jsonl | error | MUTATED FROM: member-register-list → Empty member name registration |
| mutated-member-special-chars.jsonl | member | MUTATED FROM: member-register-list → Chinese character name (张三) |
| mutated-member-label-special-chars.jsonl | member | MUTATED FROM: member-labels-types → Special characters in labels (前端@#$％) |
| discovered-conflicting-member-names.jsonl | error | DISCOVERED: Duplicate member registration — rationale: Tests duplicate detection |
| discovered-member-missing-fields.jsonl | error | DISCOVERED: Missing member name — rationale: Tests validation of required fields |
| discovered-member-label-edge-cases.jsonl | member | DISCOVERED: Empty/malformed labels — rationale: Tests label handling edge cases |
| discovered-member-type-validation.jsonl | error | DISCOVERED: Invalid member type — rationale: Tests type validation |
| discovered-member-case-sensitivity.jsonl | member | DISCOVERED: Case sensitivity in names — rationale: Tests name matching policy |

## Multi-Turn (2+ messages)

| Fixture | Turns | Category | Description |
|---------|-------|----------|-------------|
| context-resolve-by-name.jsonl | 3 | context | Create task, update by name, list |
| context-implicit-reference.jsonl | 4 | context | Create two tasks, "第一个任务" reference, list |
| workflow-create-dep-archive.jsonl | 5 | workflow | Full lifecycle: create → dep → archive → list |
| workflow-create-assign-list.jsonl | 4 | workflow | Create → register member → assign → list |
| context-cross-language.jsonl | 3 | context | Chinese create, English update by reference |
| context-error-recovery.jsonl | 3 | context | Create → invalid op → continue normally |
| workflow-dependency-add-remove.jsonl | 5 | workflow | Create two → add dep → remove dep → list |
| workflow-create-comment-list.jsonl | 3 | workflow | Create task → add comment → get detail |
| workflow-register-assign-multi.jsonl | 6 | workflow | Register 2 members → create 2 tasks → assign each → list |

## Fixture Lifecycle

- **Round 0**: Initial set (this file)
- **Round N**: Subagents MUTATE (copy + edit) or DISCOVER (new file) → update this INDEX
- Fixtures are git-tracked eval artifacts in `.eval/`
