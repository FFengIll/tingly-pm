# Round 2/2 - Part 1 Experiments

**Date:** 2026-03-28
**Method:** v2 Parallel Fuzzing
**Exploration Seed:** "留意人员配置" (Pay attention to personnel configuration)

---

## Baseline Analysis

**Pass Rate:** 17/17 (100%)

**Identified Gaps:**
1. **Member search by labels** - No way to find members by capability labels
2. **Member update** - No way to update member labels or type
3. **Member removal** - No way to remove/deactivate members
4. **Assignee validation** - When assigning tasks, the system doesn't validate the assignee exists

---

## Hypotheses

### Exp A: Add Member Search by Labels

**Hypothesis:** Adding a search_members tool with label filtering will enable users to find qualified members for task assignment.

**Target:** `tools/tools.go`, `board/member.go`

**Expected:** Users can search for members by capability labels (e.g., "find members with frontend skills")

---

### Exp B: Add Member Update Functionality

**Hypothesis:** Adding update_member tool will allow users to modify member details (labels, type) without re-registering.

**Target:** `tools/tools.go`, `board/member.go`

**Expected:** Users can update member labels and type after registration

---

### Exp C: Add Member Removal/Deactivation

**Hypothesis:** Adding remove_member tool will allow users to remove members who are no longer on the team.

**Target:** `tools/tools.go`, `board/member.go`

**Expected:** Users can remove members; removed members are no longer listed

---

## Experiment Reports

### Experiment A - Member Search by Labels

**Hypothesis:** Adding search_members tool enables finding qualified members

**Build:** PASS
**Tests:** PASS
**Result:** PASS

**Output:**
```json
{"content":"已成功注册成员 alice（human），标签为：frontend, react。","role":"assistant"}
{"content":"已成功注册成员 bob（human），标签为：backend, python。","role":"assistant"}
{"content":"找到 1 名具备 `react` 标签的成员：  \n- **alice**（human），标签：`frontend`, `react`","role":"assistant"}
```

**Observations:** The search_members tool was successfully implemented. The agent correctly registered members, found alice when searching for "react" label, and fuzzy matching works with case-insensitive substring matching.

---

### Experiment B - Member Update

**Hypothesis:** Adding update_member tool allows modifying member details

**Build:** PASS
**Tests:** PASS
**Result:** PASS

**Output:**
```json
{"content":"已成功注册成员 alice（human），标签为：frontend, react。","role":"assistant"}
{"content":"已更新成员 alice 的标签为：backend, python（类型仍为 human）。","role":"assistant"}
{"content":"当前注册的 human 成员有：\n\n- alice（human）［backend, python］\n\n如需查看 agent 类型成员，请告知。","role":"assistant"}
```

**Observations:** Successfully added UpdateMember function. Functional test confirmed that alice's labels were successfully updated from "frontend,react" to "backend,python". The members.json file confirms the final state.

---

### Experiment C - Member Removal

**Hypothesis:** Adding remove_member tool allows removing members

**Build:** PASS
**Tests:** PASS
**Result:** PASS

**Output:**
```json
{"content":"已成功注册成员 alice（human），标签为：frontend, react。","role":"assistant"}
{"content":"已成功注册成员 bob（human），标签为：backend, python。","role":"assistant"}
{"content":"已成功删除成员 alice。","role":"assistant"}
{"content":"当前注册的 human 成员有：\n\n- bob（backend, python）\n\n（alice 已被删除）","role":"assistant"}
```

**Observations:** Successfully added RemoveMember function. alice was successfully removed from members.json, timeline event "member_removed" was correctly logged. Only bob remains in the member registry as expected.

---

## Evaluation

All 3 experiments PASSED:
- **Exp A (search_members):** Enables finding members by capability labels with fuzzy matching
- **Exp B (update_member):** Enables updating member type and labels without re-registration
- **Exp C (remove_member):** Enables removing members from the registry

**Winner Selection:** All 3 experiments will be applied. They complement each other and provide a complete member lifecycle management (CRUD operations).

**Status:** Changes have already been applied by the experiment subagents. Ready for Part 2 verification.
