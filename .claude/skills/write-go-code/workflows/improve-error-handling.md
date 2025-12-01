# Workflow: Improve Error Handling

<required_reading>
**Read these reference files NOW:**
1. references/error-handling.md
2. references/effective-go.md
</required_reading>

<process>

<step name="audit-current-errors">
**Find all error handling in the code:**

Look for:
- `if err != nil` blocks
- Ignored errors (`_, err := ...` then not using err, or `_ = doSomething()`)
- Bare returns of errors without context
- `panic` calls
- Error string formatting

```bash
# Find error-related patterns
grep -n "err != nil" *.go
grep -n "_ :=" *.go
grep -n "panic(" *.go
```
</step>

<step name="fix-ignored-errors">
**Handle all errors explicitly:**

```go
// Bad: ignoring error
data, _ := json.Marshal(obj)

// Good: handle or propagate
data, err := json.Marshal(obj)
if err != nil {
    return fmt.Errorf("marshaling object: %w", err)
}

// If truly safe to ignore, document why:
_ = conn.Close() // best-effort cleanup, error doesn't affect outcome
```

Every ignored error should have a comment explaining why.
</step>

<step name="add-context">
**Wrap errors with context:**

Each layer should add context about what it was doing:

```go
// Bad: no context (hard to debug)
func (s *Service) ProcessOrder(id string) error {
    order, err := s.repo.Get(id)
    if err != nil {
        return err  // caller has no idea what failed
    }
    // ...
}

// Good: context at each level
func (s *Service) ProcessOrder(id string) error {
    order, err := s.repo.Get(id)
    if err != nil {
        return fmt.Errorf("getting order %s: %w", id, err)
    }

    if err := s.validate(order); err != nil {
        return fmt.Errorf("validating order %s: %w", id, err)
    }
    // ...
}
```

Use `%w` to wrap errors (enables `errors.Is` and `errors.As`).
</step>

<step name="define-sentinel-errors">
**Create sentinel errors for expected conditions:**

```go
package user

import "errors"

var (
    // ErrNotFound is returned when a user doesn't exist.
    ErrNotFound = errors.New("user not found")

    // ErrDuplicate is returned when creating a user that already exists.
    ErrDuplicate = errors.New("user already exists")

    // ErrInvalidEmail is returned for malformed email addresses.
    ErrInvalidEmail = errors.New("invalid email format")
)
```

Use sentinel errors when:
- Callers need to handle specific conditions differently
- The error represents a domain concept
- You want stable error comparison across versions
</step>

<step name="implement-error-checking">
**Use errors.Is and errors.As:**

```go
// Check for specific error
if errors.Is(err, user.ErrNotFound) {
    return http.StatusNotFound, "User not found"
}

// Extract error details
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    log.Printf("operation %s failed on %s", pathErr.Op, pathErr.Path)
}

// Don't do this (breaks with wrapped errors):
if err == user.ErrNotFound {  // Bad
if err.Error() == "user not found" {  // Very bad
```
</step>

<step name="format-error-strings">
**Follow error string conventions:**

```go
// Good: lowercase, no punctuation, no "error:" prefix
return errors.New("connection refused")
return fmt.Errorf("reading config: %w", err)

// Bad: capitalized, punctuation, redundant prefix
return errors.New("Error: Connection refused.")
return fmt.Errorf("Failed to read config: %w", err)
```

Error strings are often concatenated: `"service: database: connection refused"` reads naturally.
</step>

<step name="replace-panics">
**Convert panics to error returns:**

```go
// Bad: panic for recoverable errors
func Parse(data []byte) *Config {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        panic(err)  // crashes the program
    }
    return &cfg
}

// Good: return error
func Parse(data []byte) (*Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    return &cfg, nil
}
```

Reserve `panic` for:
- Truly unrecoverable situations (programmer error, impossible state)
- Package initialization that can't proceed
- Test assertions
</step>

<step name="verify">
**Test error handling:**

```go
func TestService_ProcessOrder_NotFound(t *testing.T) {
    svc := NewService(fakeRepo{err: ErrNotFound})

    err := svc.ProcessOrder("123")

    if !errors.Is(err, ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

Test that:
- Errors are returned (not swallowed)
- Error wrapping preserves the cause
- Sentinel errors are used correctly
</step>

</process>

<anti_patterns>
**Avoid:**
- Comparing error strings (`err.Error() == "..."`)
- Using `panic` for expected errors
- Logging and returning the same error (double logging)
- Creating new error types when sentinel errors suffice
- Wrapping errors without adding context (`fmt.Errorf("%w", err)`)
- Error messages starting with "Error:" or "Failed to"
</anti_patterns>

<success_criteria>
Error handling is good when:
- [ ] No errors are silently ignored
- [ ] All errors have context about what operation failed
- [ ] `fmt.Errorf` uses `%w` for wrapping
- [ ] Sentinel errors exist for conditions callers handle
- [ ] `errors.Is` / `errors.As` are used for checking
- [ ] Error strings are lowercase without punctuation
- [ ] `panic` is only used for unrecoverable situations
- [ ] Error paths are tested
</success_criteria>
