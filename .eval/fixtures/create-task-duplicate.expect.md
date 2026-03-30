# Expectations: Create Task Duplicate

## Turn 1: Create task "implement OAuth2"
- **PASS**: Calls CreateTask with title containing "OAuth2". Returns task ID.
- **FAIL**: Wrong tool or error.

## Turn 2: Create identical task "implement OAuth2"
- **PASS**: Agent detects duplicate and refuses to create a second task. References existing task.
- **FAIL**: Creates a second task (no dedup detected), or hallucinates that a task exists when it doesn't.

CONFIDENCE: high — Core smoke test for dedup detection
