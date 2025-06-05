---
description: 
globs: 
alwaysApply: false
---
When committing and pushing changes after an issue is successfully closed and tests have passed:

1.  **Get current Git branch:**
    *   Run `git rev-parse --abbrev-ref HEAD` using `run_terminal_cmd` to get the current branch name. Let this be `current_branch`.
2.  Let the closed issue number be `issue_num` and its title be `issue_title`.
3.  **Branch Management:**
    *   Set `target_branch` to `current_branch`.
    *   If `current_branch` is `main` or `master`:
        *   Generate a new branch name. The suggested pattern is `<type>/issue-<issue_num>-<slugified_issue_title>`, where `<type>` could be `feature`, `fix`, `refactor`, etc., based on the nature of the work. I will generate a slug from the issue title (e.g., lowercase, hyphens for spaces, remove special characters). Example: `refactor/issue-37-getpodnameforportforward-context-handling`.
        *   Run `git checkout -b <new_branch_name>` using `run_terminal_cmd`.
        *   Set `target_branch` to this `<new_branch_name>`.
        *   Inform the user that a new branch `<new_branch_name>` has been created and checked out.
4.  **Stage changes:** Run `git add .` using `run_terminal_cmd`.
5.  **Commit changes:**
    *   Construct a commit message. The suggested pattern is: `<CommitType>: <issue_title> (closes #<issue_num>)`. Example: `Refactor: Refactor getPodNameForPortForward for clarity, scope, and context handling (closes #37)`. The `<CommitType>` (e.g., Refactor, Fix, Feat) should match the nature of the work.
    *   Run `git commit -m "<commit_message>"` using `run_terminal_cmd`.
6.  **Push changes:** Run `git push origin <target_branch>` using `run_terminal_cmd`.
7.  Inform the user that the changes have been committed to `<target_branch>` and pushed to origin.
