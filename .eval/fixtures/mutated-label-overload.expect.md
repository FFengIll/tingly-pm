# Expectations: Label Overload

## Turns 1-2: Register members with many labels
- **PASS**: Calls UpsertMember for alice (6 labels) and bob (6 labels). All labels preserved.
- **FAIL**: Wrong tool, truncated labels, or error.

## Turns 3-5: Create tasks with many labels
- **PASS**: Calls CreateTask for each task with all specified labels.
- **FAIL**: Wrong tool, truncated labels, or error.

## Turns 6-8: Assign tasks to members
- **PASS**: Calls UpdateTask for each assignment. Correct member resolution.
- **FAIL**: Wrong tool or fails to resolve.

## Turn 9: List human-type members
- **PASS**: Calls ListMembers with type=human. Shows alice only.
- **FAIL**: Wrong tool or wrong filter.

## Turn 10: List agent-type members
- **PASS**: Calls ListMembers with type=agent. Shows bob only.
- **FAIL**: Wrong tool or wrong filter.

## Turn 11: List frontend-related tasks
- **PASS**: Calls SearchTasks or ListTasks. Returns "登录页面" (has "ui" label).
- **FAIL**: Wrong tool or no results.

## Turn 12: List backend-related tasks
- **PASS**: Calls SearchTasks or ListTasks. Returns "API接口" and "数据库设计".
- **FAIL**: Wrong tool or no results.

CONFIDENCE: high — Tests label handling with many labels and filtering
