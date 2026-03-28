# Expected Behavior: Member Label Edge Cases

## Scenario
User attempts to register a member with empty or malformed labels.

## Expected Behavior
- Agent should handle empty label list gracefully
- Should not crash or create invalid entries
- May register member without labels or treat as optional field
- Labels should be validated (non-empty strings if provided)

## Success Criteria
- Registration completes (with or without labels)
- No crash or error
- Member entry is valid in members.json
