# mutated-cross-lang-error-injection

TURN 1: task "数据迁移" created in Chinese, p0 priority
TURN 2: ERROR injection - attempts to delete nonexistent task, should gracefully fail
TURN 3: continues in English, resolves "the first task" as "数据迁移" (no ID asked), updates priority to p2
TURN 4: lists tasks showing "数据迁移" with p2 priority
