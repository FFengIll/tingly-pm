# Expectations: Rapid Tool Chaining

## Turn 1: Register member alice
- Expect: Member alice registered successfully
- Tool: MemberCreate or similar

## Turn 2: Register member bob
- Expect: Member bob registered successfully
- Tool: MemberCreate

## Turn 3: Register member charlie
- Expect: Member charlie registered successfully
- Tool: MemberCreate

## Turn 4: Create task A (任务1, p0)
- Expect: Task created with ID like TASK-YYYYMMDD-XXXXXX
- Tool: TaskCreate

## Turn 5: Create task B (任务2, p1)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 6: Create task C (任务3, p2)
- Expect: Task created with ID
- Tool: TaskCreate

## Turn 7: Assign task1 to alice
- Expect: Task assigned successfully
- Tool: TaskUpdate

## Turn 8: Assign task2 to bob
- Expect: Task assigned successfully
- Tool: TaskUpdate

## Turn 9: Assign task3 to charlie
- Expect: Task assigned successfully
- Tool: TaskUpdate

## Turn 10: List all members
- Expect: Shows alice, bob, charlie
- Tool: MemberList

## Turn 11: List all tasks
- Expect: Shows all 3 tasks with correct assignments
- Tool: TaskList

CONFIDENCE: high - Tests rapid sequential tool invocation without ambiguity