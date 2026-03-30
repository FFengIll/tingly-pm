# Mutated: Reordered Comment Detail

**Source:** workflow-create-comment-list.jsonl
**Mutation:** Reorder sequence — request task detail BEFORE creating it, then create, comment, and detail again
**Turns:** 4

## Expected Behavior

### Turn 1: View nonexistent task "安全审计"
- Agent should gracefully handle the error — task does not exist yet
- Should NOT hallucinate a task ID or details
- Should inform the user the task was not found
- **PASS criteria:** Error message, no fabricated data

### Turn 2: Create task "安全审计" with p0
- Agent should create the task successfully
- Should use CreateTask tool with correct title and priority p0
- Should confirm creation with generated task ID
- **PASS criteria:** Task created with correct title and priority

### Turn 3: Add comment "需要在下周前完成" to 安全审计
- Agent should resolve the task reference to the just-created task
- Should add the comment successfully
- **PASS criteria:** Comment added to correct task

### Turn 4: View task "安全审计" details again
- Agent should now find the task and show details including the comment
- Details should include: title, priority (p0), status (todo), the comment
- **PASS criteria:** Full details shown with comment visible

## Overall PASS criteria
- Error handling on turn 1 is graceful (no crash, no hallucination)
- Subsequent turns work correctly after the error recovery
- Data consistency: comment added in turn 3 is visible in turn 4 details
