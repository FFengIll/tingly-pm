# Expected Behavior: Member Missing Fields

## Scenario
User attempts to register a member without providing a name.

## Expected Behavior
- Agent should detect that member name is missing
- Registration should fail with clear error message
- Agent should prompt user to provide a member name
- No incomplete member entry should be created

## Success Criteria
- Registration fails appropriately
- Error message indicates missing name
- No member is registered with empty/null name
