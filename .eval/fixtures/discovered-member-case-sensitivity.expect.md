# Expected Behavior: Member Case Sensitivity

## Scenario
User registers "alice" then attempts to register "Alice" (different case).

## Expected Behavior
- Agent should determine if member names are case-sensitive
- If case-insensitive: Second registration should fail as duplicate
- If case-sensitive: Both members should be allowed
- Behavior should be consistent

## Success Criteria
- Both registrations execute without crashes
- Behavior is consistent (either both allowed or second rejected)
- Members list reflects the policy (case-sensitive or not)
