# discovered-cross-language-dedup

**Focus area:** Cross-language duplicate detection — tests whether the agent recognizes that a Chinese task title ("修复登录bug") and its English translation ("fix the login bug") describe the same task.

**Rationale:** The `create-task-duplicate` fixture tests same-language dedup, and `context-cross-language` tests cross-language reference resolution, but no fixture tests cross-language dedup. The underlying `CreateTask` tool has no server-side dedup — this relies entirely on the LLM's semantic understanding. The `SearchTasks` search uses substring matching, not semantic matching, so cross-language search would also fail at the tool level.

## Per-turn expectations

### Turn 1 (Chinese task creation)
- Tool call: `CreateTask` with title containing "修复登录bug" (or close variant), priority p1
- Should NOT hallucinate fields like assignee, description, or labels unless explicitly prompted
- Response should confirm task creation, include the generated task ID
- Response language: Chinese (matches user's language)

### Turn 2 (English duplicate attempt — KEY BEHAVIOR)
- **Ideal behavior (PASS):** Agent detects semantic duplicate, refuses to create, references the existing task from turn 1
- **Acceptable behavior (PASS with note):** Agent searches first (QueryTasks), finds the Chinese task, informs user
- **Failure modes:**
  - Creates a second task (no dedup detected) — FAIL
  - Hallucinates that a task already exists when it doesn't — FAIL
- Response language: English (matches user's language for this turn)

### Turn 3 (list verification)
- Tool call: `SearchTasks` (list mode) to show all tasks
- Should show exactly ONE task (the Chinese one from turn 1)
- If agent created a duplicate in turn 2, two tasks will appear — that's a FAIL signal

## Grading notes
- Cross-language dedup is a known capability gap — medium confidence that current agent passes
- If agent creates the duplicate, this reveals the gap for improvement
- If agent detects it, note the mechanism (LLM semantic understanding vs. search)
