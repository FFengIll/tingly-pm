# member-error-scenarios

**DISCOVERED: Member error handling scenarios — rationale: No existing fixtures test duplicate member registration errors or assigning tasks to non-existent members**

TURN 1: member "alice" registered successfully
TURN 2: attempts to register "alice" again → should return error "member alice already exists"
TURN 3: creates task "测试任务" and attempts to assign to "bob" (who doesn't exist) → should handle gracefully with error about non-existent member or auto-register behavior

CONFIDENCE: medium (agent behavior for non-existent assignee needs validation)