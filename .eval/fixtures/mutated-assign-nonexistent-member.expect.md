# Expectations: Assign to Nonexistent Member

## Turn 1: Create task "代码审查" (p1)
- **PASS**: Calls CreateTask with title containing "代码审查", priority=p1.
- **FAIL**: Wrong tool or error.

## Turn 2: Assign "代码审查" to nonexistent member charlie
- **PASS**: Agent detects that charlie is not registered. Either warns user and proceeds with the assignment anyway (assignee field set), or asks user to register the member first. Does NOT crash.
- **FAIL**: Agent crashes, hallucinates member, or produces garbled output.

CONFIDENCE: high — Tests graceful handling of nonexistent member reference
