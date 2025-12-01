---
name: write-go-code
description: Write idiomatic Go code following Effective Go and SOLID principles. Use when writing new Go code, refactoring existing code, reviewing for best practices, or improving code quality. Covers naming, interfaces, error handling, concurrency, testing, and project structure.
---

<essential_principles>
These principles apply to ALL Go code. Internalize them.

<principle name="interfaces-enable-everything">
**Interfaces Are Central to Good Go Design**

Interfaces let you apply SOLID principles to Go programs. They enable:
- Substitution (LSP) - implicit satisfaction means any type with matching methods works
- Segregation (ISP) - small interfaces (ideally single-method) reduce coupling
- Dependency inversion (DIP) - depend on abstractions, not concrete types

**The Golden Rule:** "Accept interfaces, return structs."
</principle>

<principle name="explicit-error-handling">
**Handle Errors Explicitly**

Go uses explicit error returns instead of exceptions. Every function that can fail returns an error as its last value.

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}
```

- Never ignore errors with `_` unless you document why
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use `errors.Is` and `errors.As` for error inspection
- Error strings: lowercase, no punctuation (they're often concatenated)
</principle>

<principle name="composition-over-inheritance">
**Composition Over Inheritance**

Go has no inheritance. Use embedding and interfaces:
- Embed types to reuse implementation
- Define small interfaces to specify behavior
- Compose types to build larger structures

```go
type ReadWriter interface {
    Reader
    Writer
}

type Server struct {
    *log.Logger  // embedded - gains all Logger methods
    config Config
}
```
</principle>

<principle name="share-by-communicating">
**Share Memory By Communicating**

Go's concurrency philosophy: "Do not communicate by sharing memory; instead, share memory by communicating."

- Use channels to pass data between goroutines
- Only one goroutine owns data at a time
- Use `sync.Mutex` only when channels aren't suitable
- Always ensure goroutines can exit (avoid leaks)
</principle>

<principle name="naming-matters">
**Names Convey Intent**

- **Packages:** short, lowercase, single-word (`http`, `json`, `sync`)
- **Exported names:** `MixedCaps`, describe what they do
- **Local variables:** short names for short scope (`i`, `r`, `ctx`)
- **Getters:** `Owner()` not `GetOwner()`
- **Interfaces:** method name + `-er` (`Reader`, `Closer`, `Stringer`)
- **No underscores** in names (except generated code)
</principle>
</essential_principles>

<intake>
**What would you like to do?**

1. Write new Go code (functions, types, packages)
2. Refactor existing code for better design
3. Review code for best practices
4. Fix or improve error handling
5. Write concurrent code (goroutines, channels)
6. Write tests
7. Structure a Go project
8. Something else

**Wait for response, then read the matching workflow and follow it.**
</intake>

<routing>
| Response | Workflow |
|----------|----------|
| 1, "write", "new", "implement", "create", "function", "type" | `workflows/write-new-code.md` |
| 2, "refactor", "improve", "redesign", "cleanup" | `workflows/refactor-code.md` |
| 3, "review", "check", "audit", "best practices" | `workflows/review-code.md` |
| 4, "error", "errors", "handling", "fix errors" | `workflows/improve-error-handling.md` |
| 5, "concurrent", "goroutine", "channel", "parallel" | `workflows/write-concurrent-code.md` |
| 6, "test", "tests", "testing", "TDD" | `workflows/write-tests.md` |
| 7, "structure", "project", "layout", "organize" | `workflows/structure-project.md` |
| 8, other | Clarify intent, then select appropriate workflow |

**After reading the workflow, follow it exactly.**
</routing>

<verification_loop>
After writing Go code, verify:

```bash
# 1. Format
gofmt -s -w .

# 2. Lint (includes vet and static analysis)
make lint

# 3. Build
make build

# 4. Test
make test
```

Report: "Build: OK | Tests: X pass | Lint: clean" or issues found.
</verification_loop>

<reference_index>
All domain knowledge in `references/`:

**Core Principles:** effective-go.md, solid-principles.md
**Code Quality:** naming-conventions.md, error-handling.md, anti-patterns.md
**Design:** interfaces.md, project-structure.md
**Concurrency:** concurrency-patterns.md
**Testing:** testing-patterns.md (standard testing and Ginkgo/Gomega, golden files, project patterns)
</reference_index>

<workflows_index>
| Workflow | Purpose |
|----------|---------|
| write-new-code.md | Write new functions, types, and packages |
| refactor-code.md | Improve existing code design |
| review-code.md | Audit code for best practices |
| improve-error-handling.md | Fix and enhance error handling |
| write-concurrent-code.md | Write safe concurrent code |
| write-tests.md | Write idiomatic Go tests |
| structure-project.md | Organize Go project layout |
</workflows_index>
