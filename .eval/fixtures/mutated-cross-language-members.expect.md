# Expectations: Cross-Language Members

## Turn 1: Register member 张三
- **PASS**: Calls UpsertMember with name=张三.
- **FAIL**: Wrong tool or error.

## Turn 2: Register member john
- **PASS**: Calls UpsertMember with name=john. Handles English input.
- **FAIL**: Wrong tool or error.

## Turn 3: Create task "测试任务"
- **PASS**: Calls CreateTask with title containing "测试任务".
- **FAIL**: Wrong tool or error.

## Turn 4: Assign "测试任务" to 张三
- **PASS**: Calls UpdateTask with assignee=张三. Correctly resolves Chinese member name.
- **FAIL**: Wrong tool or fails to resolve member name.

## Turn 5: List all tasks
- **PASS**: Calls ListTasks. Shows the task with 张三 as assignee.
- **FAIL**: Wrong tool or missing assignment data.

CONFIDENCE: high — Tests cross-language member name handling
