# Expected Behavior: Conflicting Member Names

## Scenario
User attempts to register the same member name twice.

## Expected Behavior
- First registration: Success - member "alice" is registered
- Second registration: Error - should detect duplicate and reject registration
- Agent should handle gracefully with clear error message
- No duplicate entries should be created in members.json

## Success Criteria
- First attempt succeeds
- Second attempt fails with appropriate error
- Only one "alice" member exists after both operations
