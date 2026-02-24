---
description: Review Go code for quality, best practices, and create issues for findings
argument-hint: [files|path|changes]
allowed-tools: Task, Bash(bd:*), Bash(git:*), Read, Grep, Glob
---

# Review Code Workflow

Review Go code for quality and create beads issues for each finding.

## Critical Rules

1. **Use `go-code-reviewer` agent** - Do not review code directly yourself
2. **Create issues for ALL findings** - Every finding must become a tracked issue
3. **Use correct priority mapping** - Critical→P0, Important→P2, Minor→P3

> **WARNING**: A code review is NOT complete until issues are created for all findings.
> Do not skip Phase 3 - every finding must become a tracked beads issue.

> **REQUIRED**: You MUST use the `go-code-reviewer` agent via the Task tool.
> Do NOT review code directly yourself - always delegate to the specialized agent.

## Phase 1: Determine Scope

Parse `$ARGUMENTS` to determine what to review:

| Argument | Action |
|----------|--------|
| Empty | Ask user what to review |
| `changes` or `diff` | Review git diff: `git diff --name-only --diff-filter=d HEAD -- '*.go'` |
| `staged` | Review staged files: `git diff --name-only --diff-filter=d --staged -- '*.go'` |
| File path(s) | Review those specific files |
| Directory path | Review all Go files in that directory |

If no Go files found, report and stop.

## Phase 2: Invoke go-code-reviewer

Use Task tool to invoke the review agent with comprehensive criteria:

```
subagent_type: "go-code-reviewer"
prompt: |
  Review the following Go files for quality issues:

  **Files to review:** [list of files from Phase 1]

  ## Review Criteria

  Evaluate the code against these 7 quality aspects:

  ### 1. Readability
  - Use clear, descriptive names for variables, functions, types, and packages
  - Follow consistent formatting and Go style conventions
  - Keep functions and methods short and focused on a single responsibility
  - Avoid clever tricks or overly condensed logic that sacrifices clarity

  ### 2. Maintainability
  - Adhere to SOLID principles (Single Responsibility, Open-Closed, Liskov Substitution, Interface Segregation, Dependency Inversion)
  - Apply DRY (Don't Repeat Yourself) by extracting repeated logic into reusable functions
  - Ensure low coupling and high cohesion: packages depend on each other minimally, related code is grouped together
  - Organize code modularly with clear separation of concerns

  ### 3. Correctness & Reliability
  - Accurately solve the problem and handle edge cases
  - Include proper error handling: validate inputs, fail gracefully, add context to errors
  - Check for comprehensive test coverage (unit tests, integration tests where appropriate)
  - Identify undefined behavior, race conditions, memory leaks, or security vulnerabilities

  ### 4. Efficiency
  - Perform well enough for the context (avoid premature optimization but eliminate obvious waste)
  - Choose appropriate algorithms and data structures
  - Scale reasonably with input growth

  ### 5. Testability
  - Design code to be easily testable (inject dependencies, prefer pure functions, avoid tight coupling to global state)
  - Check for high test coverage on critical paths
  - Identify code that is hard to test and suggest improvements

  ### 6. Documentation & Context
  - Check for clear comments explaining *why* non-obvious code exists
  - Verify docstrings/comments for exported functions, types, and packages
  - Identify missing documentation on public interfaces

  ### 7. Security & Best Practices
  - Check for input sanitization, use of safe APIs, and least-privilege principles
  - Verify adherence to Go idioms and ecosystem conventions
  - Identify OWASP top 10 vulnerabilities (injection, XSS, etc.)

  ## Output Format

  Categorize all findings by severity (Critical, Important, Minor) with:
  - `file.go:line` reference
  - Issue description
  - Specific fix suggestion

  Provide a summary count at the end.
```

Capture the review output with all findings.

## Phase 3: Create Issues (MANDATORY)

**DO NOT SKIP THIS PHASE. A review without issues is incomplete.**

Create a beads issue for EACH finding. This is non-negotiable.

### Priority Mapping

| Severity | Type | Priority | Description |
|----------|------|----------|-------------|
| Critical | bug | 0 | Bugs, races, security problems, resource leaks |
| Important | task | 2 | Design problems, missing error handling, interface violations |
| Minor | task | 3 | Naming, documentation, style improvements |

### Issue Creation

For each finding, run:

```bash
bd create --title="[Severity] Brief description" --type=<type> --priority=<priority>
```

Include in the issue title:
- Severity prefix in brackets: `[Critical]`, `[Important]`, or `[Minor]`
- Brief description of the issue
- File reference if space allows

**Group related minor findings** when they are in the same file and address the same concern.

## Phase 4: Output Summary

Provide a summary:

```
## Review Summary

**Scope:** [list of files reviewed]

**Findings:**
- Critical: X issues
- Important: Y issues
- Minor: Z issues

**Issues Created:**
- [issue-id]: [title]
- [issue-id]: [title]
...

**Run `bd list --status=open` to see all created issues.**
```

## Error Handling

| Scenario | Action |
|----------|--------|
| No arguments and unclear scope | Ask user what to review |
| No Go files found | Report "No Go files found" and stop |
| Agent fails | Report error, ask how to proceed |
| `bd create` fails | Report error, continue with remaining issues |
