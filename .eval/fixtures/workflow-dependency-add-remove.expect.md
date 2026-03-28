# workflow-dependency-add-remove

TURN 1: task "前端重构" created, p0 priority
TURN 2: task "后端优化" created, p1 priority
TURN 3: dependency added — "后端优化" blocked by "前端重构", resolves by name
TURN 4: dependency removed — "后端优化" no longer blocked
TURN 5: lists tasks showing both tasks with no blocker relationship