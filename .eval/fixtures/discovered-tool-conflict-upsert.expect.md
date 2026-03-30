# Expectations: Upsert Member Conflict

## Turn 1: Register alice as human with label 前端
- **PASS**: Calls UpsertMember with name=alice, type=human, labels containing "前端".
- **FAIL**: Wrong tool or error.

## Turn 2: Update alice's type to agent
- **PASS**: Calls UpsertMember with name=alice, type=agent. Should update existing member.
- **FAIL**: Wrong tool, or fails to update existing member.

## Turn 3: Register alice as human again
- **PASS**: Calls UpsertMember with name=alice, type=human. Updates existing member (upsert semantics).
- **FAIL**: Creates duplicate entry, or errors out.

## Turn 4: Search members by label 前端
- **PASS**: Calls SearchMembers or ListMembers with filter. Returns alice.
- **FAIL**: Wrong tool or returns wrong results.

## Turn 5: List all members
- **PASS**: Calls ListMembers. Shows alice with current type=human.
- **FAIL**: Shows stale data, or wrong tool.

CONFIDENCE: high — Tests UpsertMember idempotency and state consistency
