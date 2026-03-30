# Expectations: Error Empty Input

## Turn 1: Empty content
- **PASS**: Agent responds gracefully without crashing. Does not make any tool calls with empty/invalid arguments.
- **FAIL**: Agent crashes, makes tool calls with empty arguments, or produces garbled output.
