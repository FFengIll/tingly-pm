# Expectations: Create Task Chinese

## Turn 1: Create task with Chinese input and priority keyword
- **PASS**: Calls CreateTask with title from the Chinese input, priority correctly mapped from keyword (紧急→p0, 重要→p1, etc.). No hallucinated fields.
- **FAIL**: Wrong tool, wrong priority mapping, or hallucinated fields.
