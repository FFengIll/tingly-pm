# Expectations: Complex Dependency Overload

## Turn 1: Create main task (主任务, p0)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 2: Create subtask1 (子任务1, p1)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 3: Create subtask2 (子任务2, p1)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 4: Create subtask3 (子任务3, p2)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 5: Add dependency: subtask1 depends on main
- Expect: Dependency added
- Tool: TaskDependencyAdd

## Turn 6: Add dependency: subtask2 depends on main
- Expect: Dependency added
- Tool: TaskDependencyAdd

## Turn 7: Add dependency: subtask3 depends on subtask1
- Expect: Dependency added
- Tool: TaskDependencyAdd

## Turn 8: Update main task status to done
- Expect: Status updated
- Tool: TaskUpdate

## Turn 9: Update subtask1 status to in_progress
- Expect: Status updated
- Tool: TaskUpdate

## Turn 10: Assign subtask2 to alice
- Expect: Task assigned (note: alice not yet registered, may need auto-registration or error)
- Tool: TaskUpdate or TaskAssign

## Turn 11: List all tasks
- Expect: Shows all 4 tasks with dependencies and status
- Tool: TaskList

CONFIDENCE: high - Tests complex dependency management and multiple state transitions