# Expectations: Tool Redundancy Check

## Turn 1: Create task "测试任务1" (p1)
- **PASS**: Calls CreateTask. Returns task ID.
- **FAIL**: Wrong tool or error.

## Turn 2: Create task "测试任务2" (p1)
- **PASS**: Calls CreateTask. Returns task ID.
- **FAIL**: Wrong tool or error.

## Turn 3: List all tasks
- **PASS**: Calls ListTasks. Shows both tasks.
- **FAIL**: Wrong tool or missing tasks.

## Turn 4: List all tasks again (redundant)
- **PASS**: Ideally avoids redundant call if nothing changed. Acceptable: calls ListTasks again since user explicitly asked.
- **FAIL**: Errors out.

## Turn 5: List all tasks a third time (redundant)
- **PASS**: Same as turn 4 — acceptable since user asked again.
- **FAIL**: Errors out.

## Turn 6: Register alice (human, frontend)
- **PASS**: Calls UpsertMember.
- **FAIL**: Wrong tool or error.

## Turn 7: Register bob (agent, planning)
- **PASS**: Calls UpsertMember.
- **FAIL**: Wrong tool or error.

## Turn 8: Assign 测试任务1 to alice
- **PASS**: Calls UpdateTask with assignee=alice.
- **FAIL**: Wrong tool or fails to resolve task/member reference.

## Turn 9: Assign 测试任务1 to alice again (redundant)
- **PASS**: Agent should recognize alice is already assigned. May skip or confirm. Should NOT produce error.
- **FAIL**: Errors out, or hallucinates.

## Turn 10: Assign 测试任务2 to bob
- **PASS**: Calls UpdateTask with assignee=bob.
- **FAIL**: Wrong tool or fails to resolve.

## Turn 11: List all tasks
- **PASS**: Calls ListTasks. Shows both tasks with correct assignments (任务1→alice, 任务2→bob).
- **FAIL**: Wrong assignments or wrong tool.

CONFIDENCE: high — Tests whether agent handles redundant user requests gracefully
