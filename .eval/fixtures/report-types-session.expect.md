# Expectations: Report Types + Session Save

## Turn 1: Create task "用户登录" (p1)
- **PASS**: Calls CreateTask with title containing "用户登录", priority=p1.
- **FAIL**: Wrong tool or error.

## Turn 2: Save session with label "before-report"
- **PASS**: Calls SaveSession with label="before-report". Returns confirmation.
- **FAIL**: Wrong tool or error.

## Turn 3: Generate weekly report
- **PASS**: Calls GenerateReport with type=weekly. Returns weekly summary.
- **FAIL**: Wrong tool, generates daily instead of weekly, or error.

## Turn 4: Generate project summary
- **PASS**: Calls GenerateReport with type=summary or Summary. Returns project stats.
- **FAIL**: Wrong tool or error.

CONFIDENCE: high — Tests report type variety and session save
