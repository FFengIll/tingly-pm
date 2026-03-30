# Expectations: Create Task English

## Turn 1: Create task "implement OAuth2"
- **PASS**: Calls CreateTask with title containing "OAuth2" (or close variant), priority=p1 (default). No hallucinated fields (assignee, description, labels).
- **FAIL**: Wrong tool, hallucinated fields, or missing task ID in response.
