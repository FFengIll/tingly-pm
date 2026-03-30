# Expectations: Update Task Single Field

## Turn 1: Create task "API integration" with p2
- **PASS**: Calls CreateTask with title containing "API integration", priority=p2.
- **FAIL**: Wrong tool, wrong priority, or hallucinated fields.

## Turn 2: Update the API integration task priority to p0
- **PASS**: Calls UpdateTask with correct task_id (resolved from "API integration") and priority=p0. Does NOT change other fields (title, status, assignee remain unchanged).
- **FAIL**: Wrong tool, fails to resolve task reference, hallucinates changes to other fields, or sets wrong priority.

CONFIDENCE: high — Core smoke test for update without hallucination
