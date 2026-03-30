# Expectations: Redundant Tool Usage (Compound Request)

## Turn 1: Register alice (human, frontend)
- **PASS**: Calls UpsertMember.
- **FAIL**: Wrong tool or error.

## Turn 2: Create task "代码审查" (p1)
- **PASS**: Calls CreateTask.
- **FAIL**: Wrong tool or error.

## Turn 3: Assign "代码审查" to alice AND set status to in_progress
- **PASS**: Makes at most 1 UpdateTask call combining both changes (assignee + status), or 2 calls if single call can't do both. No redundant list/query calls before updating.
- **FAIL**: Makes 3+ tool calls (list/search before update when not needed), wrong tool, or hallucinates.

CONFIDENCE: medium — Tests tool efficiency with compound user request
