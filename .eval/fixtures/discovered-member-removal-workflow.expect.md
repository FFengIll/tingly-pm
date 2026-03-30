# DISCOVERED: Member removal workflow — rationale: remove_member tool is never tested in any existing fixture. Tests full lifecycle of member removal including verifying member is gone after removal, and that previously assigned tasks still exist. CONFIDENCE: high

## Turn 1: Register member bob
- **PASS**: Calls UpsertMember with name=bob, type=human, labels=["backend"]. No hallucinated fields.
- **FAIL**: Wrong tool, hallucinated fields, or missing required fields.

## Turn 2: Create task "API接口重构" assigned to bob
- **PASS**: Calls CreateTask with title="API接口重构", priority=p0, assignee=bob. Confirms creation with task ID.
- **FAIL**: Wrong priority mapping, wrong tool, hallucinated task ID, or fails to find member.

## Turn 3: Create task "数据库优化" assigned to bob
- **PASS**: Calls CreateTask with title="数据库优化", priority=p1, assignee=bob. Confirms creation with task ID.
- **FAIL**: Wrong priority, hallucinated data, or duplicate detection incorrectly triggers.

## Turn 4: List bob's tasks
- **PASS**: Calls SearchTasks or ListTasks with assignee filter for bob. Returns 2 tasks (API接口重构, 数据库优化).
- **FAIL**: Returns wrong count, fails to filter by member, or calls wrong tool.

## Turn 5: Remove member bob
- **PASS**: Calls RemoveMember with name=bob. Confirms removal.
- **FAIL**: Calls wrong tool (e.g., UpsertMember), hallucinates output, or fails with error.

## Turn 6: List all members
- **PASS**: Calls ListMembers. Bob should NOT appear in results. Only shows remaining members (none, since only bob was registered).
- **FAIL**: Bob still appears, or agent hallucinates members.

## Turn 7: List all tasks
- **PASS**: Calls ListTasks. Both tasks (API接口重构, 数据库优化) should still exist even though bob was removed. Tasks show bob as assignee (or empty assignee, depending on implementation).
- **FAIL**: Tasks are missing, or agent incorrectly reports that tasks were deleted.
