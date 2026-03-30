# Expectations: Assign to Empty/Invalid Member

## Turn 1: Create task "代码审查" (p1)
- **PASS**: Calls CreateTask with title containing "代码审查", priority=p1.
- **FAIL**: Wrong tool or error.

## Turn 2: Assign task to empty/invalid member
- **PASS**: Agent handles gracefully — asks for clarification, reports error, or refuses. Does NOT crash.
- **FAIL**: Agent crashes, or assigns to empty string without validation.

CONFIDENCE: medium — Tests error handling for invalid assignment
