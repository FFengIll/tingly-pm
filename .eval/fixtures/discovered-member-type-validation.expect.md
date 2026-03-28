# Expected Behavior: Member Type Validation

## Scenario
User attempts to register a member with an invalid type.

## Expected Behavior
- Agent should validate member type (should be "human" or "agent")
- Invalid type should be rejected with clear error
- Agent may suggest valid types
- No member with invalid type should be created

## Success Criteria
- Registration fails for invalid type
- Error message indicates type validation issue
- Member is not registered with invalid type
