# Expectations: Create Task + Assign

## Turn 1: Register member alice
- **PASS**: Calls UpsertMember with name=alice, type=human. No hallucinated fields.
- **FAIL**: Wrong tool or error.

## Turn 2: Create task "代码审查" and assign to alice
- **PASS**: Calls CreateTask with title containing "代码审查", assignee=alice. Single tool call is acceptable; calling CreateTask then UpdateTask is also acceptable.
- **FAIL**: Fails to register alice reference, wrong tool, or hallucinated data.

CONFIDENCE: high — Core smoke test for create + assign workflow
