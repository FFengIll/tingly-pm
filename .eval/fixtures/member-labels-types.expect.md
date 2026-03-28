# member-labels-types

**DISCOVERED: Member capability labels and type filtering — rationale: No existing fixtures test the labels parameter for capability tagging or type-based member filtering**

TURN 1: member "alice" registered as human with labels ["前端", "react"]
TURN 2: member "bob" registered as agent with labels ["后端", "python"]
TURN 3: lists only human members (shows alice)
TURN 4: lists only agent members (shows bob)
TURN 5: task "开发登录页面" created and assigned to alice
TURN 6: task "API开发" created and assigned to bob
TURN 7: lists all tasks showing both assignments with correct member types

CONFIDENCE: high