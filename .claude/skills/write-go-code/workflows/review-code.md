# Workflow: Review Go Code

<required_reading>
**Read these reference files NOW before reviewing:**
1. references/effective-go.md
2. references/anti-patterns.md
3. references/naming-conventions.md
</required_reading>

<process>

<step name="check-formatting">
**Verify formatting and linting:**

```bash
# Check formatting
gofmt -d .

# Run linters (includes vet and static analysis)
make lint
```

All Go code should pass `gofmt` with no changes.
</step>

<step name="review-package-design">
**Evaluate package organization:**

Questions to ask:
- Does the package name describe its purpose?
- Is there a single, clear responsibility?
- Are dependencies reasonable? (check imports)
- Is the public API minimal and focused?

Red flags:
- Package names: `utils`, `common`, `helpers`, `misc`
- Import cycles (won't compile, but check design)
- Huge packages (consider splitting)
- Many exported types/functions (consider what's really public)
</step>

<step name="review-naming">
**Check naming conventions:**

| Item | Convention | Example |
|------|------------|---------|
| Package | lowercase, short | `user`, `http` |
| Exported type | MixedCaps, noun | `UserStore`, `Config` |
| Unexported type | mixedCaps, noun | `requestHandler` |
| Interface | MixedCaps, -er suffix | `Reader`, `Stringer` |
| Function | MixedCaps, verb | `GetUser`, `Parse` |
| Getter | No Get prefix | `u.Name()` not `u.GetName()` |
| Local var | Short for small scope | `i`, `r`, `ctx`, `err` |
| Receiver | 1-2 letter abbreviation | `(s *Server)`, `(u *User)` |

Check for:
- Stuttering: `user.UserStore` should be `user.Store`
- Underscores in names (avoid except in test files)
- Overly long names for small scopes
- Unclear abbreviations
</step>

<step name="review-error-handling">
**Audit error handling:**

Check every error return:
- Is it handled? (not ignored with `_`)
- Is context added when wrapping? (`fmt.Errorf("doing X: %w", err)`)
- Are sentinel errors used appropriately?
- Are error strings lowercase without punctuation?

Red flags:
```go
// Bad: ignored error
result, _ := doSomething()

// Bad: no context
if err != nil {
    return err
}

// Bad: error string formatting
return fmt.Errorf("Failed to process: %w.", err)

// Good
if err != nil {
    return fmt.Errorf("processing request: %w", err)
}
```
</step>

<step name="review-interfaces">
**Evaluate interface design:**

Good interfaces:
- Small (1-2 methods ideal)
- Defined where used, not where implemented
- Named with -er suffix for single-method interfaces
- Describe behavior, not data

Red flags:
```go
// Bad: too many methods
type UserManager interface {
    Get(id string) (*User, error)
    Save(u *User) error
    Delete(id string) error
    List() ([]*User, error)
    Count() int
    Validate(u *User) error
    // ... more methods
}

// Better: split by use case
type UserGetter interface {
    Get(id string) (*User, error)
}
```
</step>

<step name="review-concurrency">
**Check concurrent code:**

If goroutines or channels are used:
- Can goroutines leak? (ensure they can exit)
- Is shared state protected?
- Are channels closed appropriately? (sender closes, not receiver)
- Is `context.Context` used for cancellation?
- Would `make test` pass (includes race detection)?

Red flags:
```go
// Bad: goroutine leak
go func() {
    for item := range ch {  // blocks forever if ch never closes
        process(item)
    }
}()

// Bad: data race
go func() {
    counter++  // unprotected write
}()
```
</step>

<step name="review-documentation">
**Check documentation:**

All exported items should have doc comments:
- Start with the name being documented
- Complete sentences
- Explain behavior, not implementation
- Document error conditions

```go
// Good
// Get retrieves a user by ID.
// It returns ErrNotFound if no user exists with the given ID.
func (s *Store) Get(id string) (*User, error)

// Bad: doesn't start with name, incomplete
// This function gets users
func (s *Store) Get(id string) (*User, error)
```
</step>

<step name="summarize-findings">
**Categorize issues:**

- **Critical:** Bugs, race conditions, resource leaks, security issues
- **Important:** Design problems, missing error handling, test gaps
- **Minor:** Naming, documentation, style

Provide specific, actionable feedback with code examples of improvements.
</step>

</process>

<anti_patterns>
**Common issues to flag:**
- Returning unexported types from exported functions
- Empty interface (`interface{}`) overuse
- Pointer to interface (almost never correct)
- `panic` for recoverable errors
- `init()` functions with side effects
- Global mutable state
- Ignoring context cancellation
- Missing `defer` for cleanup
</anti_patterns>

<success_criteria>
A thorough review covers:
- [ ] Formatting and linting pass
- [ ] Package design is appropriate
- [ ] Naming follows conventions
- [ ] Errors are handled properly
- [ ] Interfaces are well-designed
- [ ] Concurrent code is safe
- [ ] Documentation exists for public API
- [ ] Issues are categorized by severity
</success_criteria>
