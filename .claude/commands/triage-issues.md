---
description: "Triage open issues: verify against code, close irrelevant, group for efficient commits, set dependencies"
allowed-tools: Task, Bash(bd:*), Read, Grep, Glob
---

# Triage Issues Workflow

Triage all open issues: verify each against the codebase, close invalid ones, group related issues for efficient commits, and wire up dependencies.

## Critical Rules

1. **No user confirmation needed for closures** - the `--reason` flag serves as the audit trail
2. **Use Explore agents for verification** - do NOT search the codebase directly; delegate to Task tool with `Explore` subagent
3. **Close aggressively** - if the code doesn't exhibit the described problem, close it
4. **Group before fixing** - issues touching the same file(s) or concern should share a single commit
5. **Use TaskCreate to track progress** - create a todo for each phase

## Phase 1: Gather Issues

Collect all open issues with full details.

```bash
bd list --sort priority
```

For each issue, fetch full details:

```bash
bd show <id> --json
```

Build a working table:

| ID | Title | Priority | Type | Description Summary |
|----|-------|----------|------|---------------------|

## Phase 2: Verify Against Codebase

For each issue, launch an **Explore agent** via the Task tool to verify whether the described problem still exists.

```
subagent_type: "Explore"
prompt: |
  Investigate whether this issue is valid by searching the codebase:

  **Issue:** [title]
  **Description:** [description]

  Search for the relevant code. Determine:
  1. Does the described problem actually exist in the current code?
  2. If it references a specific file/function, does that file/function exist?
  3. Has the problem already been fixed?
  4. Is this by-design behavior or defensive coding that should stay?

  Report: file:line references found, whether the problem exists, and your confidence level.
```

**Launch multiple Explore agents in parallel** when there are many issues — batch up to 4 at a time.

### Classification

After verification, classify each issue:

| Classification | Criteria | Action |
|----------------|----------|--------|
| **Valid** | Problem confirmed in code, worth fixing | Keep open |
| **Invalid** | Problem doesn't exist, already fixed, or file/function not found | Close immediately |
| **Not worth fixing** | By-design, defensive coding, trivial impact, or cost exceeds benefit | Close immediately |

## Phase 3: Close Irrelevant Issues

Close all issues classified as **invalid** or **not worth fixing**:

```bash
bd close <id> --reason="<explanation of why this is being closed>"
```

### Reason Templates

| Classification | Reason Format |
|----------------|---------------|
| Already fixed | `"Already fixed: <brief explanation of current state>"` |
| Doesn't exist | `"Invalid: <thing referenced> does not exist in the codebase"` |
| By design | `"By design: <why current behavior is intentional>"` |
| Defensive coding | `"Not worth fixing: defensive coding that prevents future regressions"` |
| Trivial | `"Not worth fixing: trivial impact, cost exceeds benefit"` |

## Phase 4: Group and Merge

Analyze remaining valid issues for grouping opportunities.

### Grouping Criteria

| Signal | Example |
|--------|---------|
| Same file(s) | Two issues both modify `internal/core/pool.go` |
| Same concern | Multiple naming/style issues across related files |
| Same fix | Issues that would be resolved by a single refactor |
| Logical unit | Feature + its test should be one commit |

### Merge Process

For each group:

1. **Pick the primary issue** — the one with the broadest scope or lowest ID
2. **Update the primary** to cover all merged work:
   ```bash
   bd update <primary-id> --title="<updated title covering merged scope>"
   ```
3. **Close absorbed issues** referencing the primary:
   ```bash
   bd close <absorbed-id> --reason="Merged into <primary-id>"
   ```

## Phase 5: Define Dependencies and Summarize

### Wire Dependencies

For issues where one must complete before another can start:

```bash
bd dep <blocker-id> --blocks <blocked-id>
```

For issues that are related but not blocking:

```bash
bd dep relate <id1> <id2>
```

### Dependency Patterns

| Pattern | Dependency |
|---------|------------|
| Refactor before feature | Refactor blocks feature |
| Bug fix enables test | Bug fix blocks test |
| Core before consumers | Core module blocks dependent modules |
| Interface before implementation | Interface design blocks implementations |

### Final Summary

Output the final state:

```bash
bd list --sort priority
bd graph --all
```

Present a summary:

```
## Triage Summary

### Closed: X issues
- [id]: [reason]
...

### Merged: Y issues into Z primaries
- [absorbed-ids] → [primary-id]: [merged title]
...

### Dependencies Added: N
- [blocker] blocks [blocked]: [reason]
...

### Remaining Valid Issues: M
[table of remaining issues sorted by priority]

### Recommended Work Order
[numbered list based on dependencies and priority]
```

## Error Handling

| Scenario | Action |
|----------|--------|
| No open issues | Report "No open issues to triage" and stop |
| `bd show` fails for an issue | Skip that issue, continue with others |
| `bd close` fails | Report error, continue with remaining closures |
| `bd dep` fails | Report error, continue with remaining dependencies |
| Explore agent inconclusive | Keep issue open (err on side of caution) |
| All issues classified as valid | Skip Phase 3, proceed to grouping |
| No grouping opportunities | Skip Phase 4, proceed to dependencies |
