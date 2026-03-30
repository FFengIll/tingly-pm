# Expectations: Rapid Dependency Add/Remove/Re-add

## Turn 1: Register alice (human, frontend, react)
- **PASS**: Calls UpsertMember with name=alice, labels containing frontend and react.
- **FAIL**: Wrong tool or error.

## Turn 2: Register bob (agent, planning)
- **PASS**: Calls UpsertMember with name=bob, type=agent.
- **FAIL**: Wrong tool or error.

## Turn 3: Create task A "前端重构" (p0)
- **PASS**: Calls CreateTask with title containing "前端重构", priority=p0.
- **FAIL**: Wrong tool or error.

## Turn 4: Create task B "测试" (p1)
- **PASS**: Calls CreateTask with title containing "测试", priority=p1.
- **FAIL**: Wrong tool or error.

## Turn 5: Add dependency: 前端重构 depends on 测试
- **PASS**: Calls AddDependency with correct task IDs. 前端重构's blocked_by should include 测试.
- **FAIL**: Wrong tool, wrong task resolution, or error.

## Turn 6: Remove dependency
- **PASS**: Calls RemoveDependency. 前端重构's blocked_by should no longer include 测试.
- **FAIL**: Wrong tool or fails to remove.

## Turn 7: Re-add dependency
- **PASS**: Calls AddDependency again. 前端重构's blocked_by should include 测试 again.
- **FAIL**: Wrong tool or error.

## Turn 8: List all tasks
- **PASS**: Calls ListTasks. Shows both tasks. 前端重构 shows dependency on 测试.
- **FAIL**: Wrong tool, missing dependency info, or incorrect state.

CONFIDENCE: high — Tests dependency add/remove cycle and state consistency
