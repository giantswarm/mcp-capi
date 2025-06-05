---
description: 
globs: 
alwaysApply: true
---
When closing an issue:

1.  Confirm with the user that the issue is ready to be closed.
2.  Make sure to run `goimports -w .` and `go fmt ./...` before your commit any code.
3.  Run tests by executing the `make test` command in the terminal using `run_terminal_cmd`.
4.  **If tests fail:**
    *   You have to fix the tests.
    *   DO NOT proceed with closing the issue.
5.  **If tests pass:**
    *   Proceed with committing and pushing the changes, follow the `commit-and-push-workflow`.
    *   After the `commit-and-push-workflow` is successful, then use the `mcp_github_update_issue` tool to change the state of the issue to 'closed'. Make sure to specify the `owner` as `giantswarm`, `repo` as `envctl`, and the correct `issue_number`.