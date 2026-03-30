# mutated-cross-lang-update

MUTATED FROM: context-resolve-by-name -> mixed language: create in Chinese, update command in English referencing Chinese task name

TURN 1: task "用户认证" created with p0 priority (Chinese input)
TURN 2: English command to update status to in_progress, references Chinese name "用户认证" -- agent must resolve by name across language boundary
TURN 3: lists tasks showing "用户认证" with in_progress status

Expected behavior: agent should handle the cross-language update correctly, resolving the Chinese task name from the English update command. Response to turn 2 should acknowledge the update. Turn 3 should confirm the status change.
