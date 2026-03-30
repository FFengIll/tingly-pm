# Expectations: Tool Ambiguity (Empty Board)

## Turn 1: Search tasks (no query term)
- **PASS**: Calls ListTasks or SearchTasks. Handles empty board gracefully ("No tasks found" or similar).
- **FAIL**: Errors out, or hallucinates tasks.

## Turn 2: List all tasks
- **PASS**: Calls ListTasks. Returns empty result gracefully.
- **FAIL**: Errors out or calls wrong tool.

## Turn 3: Generate daily report
- **PASS**: Calls GenerateReport. Handles empty board gracefully.
- **FAIL**: Errors out or calls wrong tool.

## Turn 4: View timeline
- **PASS**: Calls GenerateReport (type=timeline) or ListTimeline. Handles empty timeline gracefully.
- **FAIL**: Errors out or calls wrong tool.

## Turn 5: Get project stats
- **PASS**: Calls GenerateReport (type=summary) or Summary. Handles empty project gracefully.
- **FAIL**: Errors out or calls wrong tool.

## Turn 6: List all members
- **PASS**: Calls ListMembers. Returns empty result gracefully.
- **FAIL**: Errors out or calls wrong tool.

CONFIDENCE: medium — Tests agent's ability to handle multiple listing/reporting tools on empty state
