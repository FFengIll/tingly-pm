# workflow-create-dep-archive

TURN 1: task "用户认证" created, p0 priority
TURN 2: task "登录页面" created, p1 priority
TURN 3: dependency added — resolves both tasks by name, "登录页面" blocked by "用户认证"
TURN 4: resolves "用户认证" by name, archives as done
TURN 5: lists tasks — "用户认证" not in active list, "登录页面" still active