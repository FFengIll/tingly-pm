# Expectations: Rapid Tool Chaining

## Turn 1: Register member alice
- **PASS**: Calls UpsertMember with name=alice. No hallucinated fields.
- **FAIL**: Wrong tool or error.

## Turn 2: Register member bob
- **PASS**: Calls UpsertMember with name=bob.
- **FAIL**: Wrong tool or error.

## Turn 3: Register member charlie
- **PASS**: Calls UpsertMember with name=charlie.
- **FAIL**: Wrong tool or error.

## Turn 4: Create task A (任务1, p0)
- **PASS**: Calls CreateTask with title containing "任务1", priority=p0.
- **FAIL**: Wrong tool, hallucinated fields, or missing required fields.

## Turn 5: Create task B (任务2, p1)
- **PASS**: Calls CreateTask with title containing "任务2", priority=p1.
- **FAIL**: Wrong tool or wrong priority.

## Turn 6: Create task C (任务3, p2)
- **PASS**: Calls CreateTask with title containing "任务3", priority=p2.
- **FAIL**: Wrong tool or wrong priority.

## Turn 7: Assign task1 to alice
- **PASS**: Calls UpdateTask to set assignee=alice on the correct task.
- **FAIL**: Wrong tool or fails to resolve task reference.

## Turn 8: Assign task2 to bob
- **PASS**: Calls UpdateTask to set assignee=bob on the correct task.
- **FAIL**: Wrong tool or fails to resolve task reference.

## Turn 9: Assign task3 to charlie
- **PASS**: Calls UpdateTask to set assignee=charlie on the correct task.
- **FAIL**: Wrong tool or fails to resolve task reference.

## Turn 10: List all members
- **PASS**: Calls ListMembers. Shows alice, bob, charlie.
- **FAIL**: Wrong tool or missing members.

## Turn 11: List all tasks
- **PASS**: Calls ListTasks. Shows all 3 tasks with correct assignments.
- **FAIL**: Wrong tool or missing/incorrect task data.

CONFIDENCE: high — Tests rapid sequential tool invocation without ambiguity
