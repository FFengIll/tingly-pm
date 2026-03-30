# Expectations: Redundant List Members

## Turn 1: Register alice (human, frontend)
- **PASS**: Calls UpsertMember.
- **FAIL**: Wrong tool or error.

## Turn 2: Register bob (agent, backend)
- **PASS**: Calls UpsertMember.
- **FAIL**: Wrong tool or error.

## Turn 3: Search members alice
- **PASS**: Calls SearchMembers or ListMembers. Returns alice.
- **FAIL**: Wrong tool or no results.

## Turn 4: List all members
- **PASS**: Calls ListMembers. Shows alice and bob.
- **FAIL**: Wrong tool or missing members.

## Turn 5: Search members bob
- **PASS**: Calls SearchMembers or ListMembers. Returns bob.
- **FAIL**: Wrong tool or no results.

## Turn 6: List all members again
- **PASS**: Calls ListMembers. Shows alice and bob.
- **FAIL**: Wrong tool or missing members.

CONFIDENCE: medium — Tests alternating search/list patterns (minor redundancy acceptable)
