---
description: 
globs: 
alwaysApply: true
---
## Testing Guidelines

When implementing new features or fixing bugs, ensure the following testing criteria are met:

- New functionality must be covered by appropriate unit, integration, or end-to-end tests.
- If the user describes a bug, the behaviour that lead to the bug needs to be covered in a test.
- Existing tests must be updated to reflect any changes in behavior.
- Never use timers, timeouts, or other means to wait for conditions in the tests. This is bad practice and only produces slow and flaky tests.
- If the changes affect the TUI views:
    - Corresponding golden files (located in `internal/tui/view/testdata/*.golden`) must be generated or updated.
    - This is typically done by running `go test ./internal/tui/view/... -update`.
    - The updated golden files must be carefully reviewed and verified to ensure they reflect the intended visual changes.
- Strive to meet or improve test coverage for the modified packages. There should be a minimum of 80% test coverage.

To run all tests and linters, use the `make test` command in your terminal. If you changed or added view tests use the terminal to execute 'NO_COLOR=true go test ./internal/tui/view/... -update' so that the view tests have been updated not to use any colors.