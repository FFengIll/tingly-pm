# Expectations: Complex Dependency Overload

## Turn 1: Register member alice
- Expect: Member registered successfully
- Tool: UpsertMember

## Turn 2: Create main task (主任务, p0)
- Expect: Task created with ID
- Tool: CreateTask

## Turn 3: Create subtask1 (子任务1, p1)
- Expect: Task created with ID
- Tool: CreateTask

## Turn 4: Create subtask2 (子任务2, p1)
- Expect: Task created with ID
- Tool: CreateTask

## Turn 5: Create subtask3 (子任务3, p2)
- Expect: Task created with ID
- Tool: CreateTask

## Turn 6: Add dependency: subtask1 depends on main
- Expect: Dependency added
- Tool: AddDependency

## Turn 7: Add dependency: subtask2 depends on main
- Expect: Dependency added
- Tool: AddDependency

## Turn 8: Add dependency: subtask3 depends on subtask1
- Expect: Dependency added
- Tool: AddDependency

## Turn 9: Update main task status to done
- Expect: Status updated
- Tool: UpdateTask

## Turn 10: Update subtask1 status to in_progress
- Expect: Status updated
- Tool: UpdateTask

## Turn 11: Assign subtask2 to alice
- Expect: Task assigned to alice (member registered in turn 1)
- Tool: UpdateTask

## Turn 12: List all tasks
- Expect: Shows all 4 tasks with dependencies and status
- Tool: ListTasks

CONFIDENCE: high - Tests complex dependency management and multiple state transitions
