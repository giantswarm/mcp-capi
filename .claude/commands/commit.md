---
description: Commit staged changes with optional message
argument-hint: [message]
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git log:*), Bash(git add:*), Bash(git commit:*)
---

Commit the currently staged changes to git.

If a commit message is provided via $ARGUMENTS, use that message directly.

If no message is provided, analyze the staged changes and generate an appropriate commit message following these rules:
1. Use imperative mood ("Add feature" not "Added feature")
2. Keep the first line under 72 characters
3. Focus on the "why" rather than the "what"
4. Be concise but descriptive

Steps:
1. Run `git status` to verify there are staged changes
2. Run `git diff --cached` to see what's staged
3. If $ARGUMENTS is provided, use it as the commit message
4. Otherwise, generate an appropriate imperative-style commit message based on the staged changes
5. Create the commit
6. Show the result with `git status`
