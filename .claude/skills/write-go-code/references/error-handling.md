<overview>
Go uses explicit error returns instead of exceptions. This makes error handling visible and forces consideration of failure modes at every step.
</overview>

<basics>
**The error interface:**

```go
type error interface {
    Error() string
}
```

**Convention: error is the last return value:**

```go
func DoSomething() (Result, error)
func Read(p []byte) (n int, err error)
```

**Always check errors:**

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}
```
</basics>

<creating_errors>
**Simple errors:**

```go
import "errors"

// Static error message
err := errors.New("something went wrong")

// Formatted error
err := fmt.Errorf("failed to process %s: invalid format", filename)
```

**Sentinel errors (for callers to check):**

```go
var (
    ErrNotFound   = errors.New("not found")
    ErrPermission = errors.New("permission denied")
    ErrTimeout    = errors.New("operation timed out")
)

// Usage
if errors.Is(err, ErrNotFound) {
    // handle not found case
}
```

**Custom error types (when you need extra data):**

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Usage
var valErr *ValidationError
if errors.As(err, &valErr) {
    fmt.Printf("field %s is invalid: %s\n", valErr.Field, valErr.Message)
}
```
</creating_errors>

<wrapping_errors>
**Add context when propagating:**

```go
// Use %w to wrap (preserves error chain)
if err != nil {
    return fmt.Errorf("processing user %s: %w", userID, err)
}

// Result: "processing user 123: fetching data: connection refused"
```

**When to wrap vs not wrap:**

| Situation | Action |
|-----------|--------|
| Adding context | Wrap with `%w` |
| Hiding implementation details | Don't wrap, create new error |
| Returning sentinel errors | Return as-is or wrap |

**Custom types with wrapping:**

```go
type RequestError struct {
    StatusCode int
    Err        error
}

func (e *RequestError) Error() string {
    return fmt.Sprintf("request failed with status %d: %v", e.StatusCode, e.Err)
}

func (e *RequestError) Unwrap() error {
    return e.Err
}
```
</wrapping_errors>

<checking_errors>
**errors.Is - check for specific error:**

```go
// Works through wrapped errors
if errors.Is(err, ErrNotFound) {
    return http.StatusNotFound
}

// Also works:
if errors.Is(err, context.Canceled) {
    return // operation was canceled
}
```

**errors.As - extract error type:**

```go
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    fmt.Printf("operation %s failed on path %s\n", pathErr.Op, pathErr.Path)
}

var netErr net.Error
if errors.As(err, &netErr) && netErr.Timeout() {
    // handle timeout specifically
}
```

**Don't do this:**

```go
// Bad: breaks with wrapped errors
if err == ErrNotFound { ... }

// Bad: fragile, depends on exact message
if err.Error() == "not found" { ... }
if strings.Contains(err.Error(), "timeout") { ... }
```
</checking_errors>

<error_strings>
**Formatting conventions:**

```go
// Good: lowercase, no punctuation
errors.New("connection refused")
fmt.Errorf("reading config: %w", err)

// Bad: capitalized, punctuation
errors.New("Connection refused.")
fmt.Errorf("Error: Reading config: %w", err)
```

**Why?** Errors are often chained:
```
"service: database: connection refused"  // reads naturally
"Service: Database: Connection refused."  // awkward
```
</error_strings>

<handling_patterns>
**Return early:**

```go
func process(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }

    result, err := parse(data)
    if err != nil {
        return fmt.Errorf("parsing: %w", err)
    }

    if err := validate(result); err != nil {
        return fmt.Errorf("validating: %w", err)
    }

    return nil
}
```

**Defer for cleanup:**

```go
func process(filename string) (err error) {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    // work with file...
    return nil
}
```

**Handle or return, never both:**

```go
// Bad: logs AND returns (double handling)
if err != nil {
    log.Printf("error: %v", err)
    return err
}

// Good: return with context, let caller decide to log
if err != nil {
    return fmt.Errorf("processing request: %w", err)
}

// Good: log and handle (don't return)
if err != nil {
    log.Printf("non-critical error: %v", err)
    // continue with fallback behavior
}
```
</handling_patterns>

<panic_recover>
**When to panic:**

```go
// Programmer error / impossible state
func mustCompile(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

// Package initialization that can't proceed
var defaultClient = mustCreateClient()
```

**When NOT to panic:**

```go
// Bad: recoverable errors
func Parse(s string) int {
    n, err := strconv.Atoi(s)
    if err != nil {
        panic(err)  // Don't do this!
    }
    return n
}

// Good: return error
func Parse(s string) (int, error) {
    return strconv.Atoi(s)
}
```

**Recover for boundaries:**

```go
// HTTP handler that shouldn't crash server
func handler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic in handler: %v", r)
            http.Error(w, "Internal Server Error", 500)
        }
    }()

    // handle request...
}
```
</panic_recover>

<testing_errors>
**Test error conditions:**

```go
func TestParse_InvalidInput(t *testing.T) {
    _, err := Parse("not a number")

    if err == nil {
        t.Fatal("expected error, got nil")
    }
}

func TestService_NotFound(t *testing.T) {
    svc := NewService(emptyStore{})

    _, err := svc.Get("nonexistent")

    if !errors.Is(err, ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```
</testing_errors>

<anti_patterns>
**Common mistakes:**

```go
// Ignoring errors
result, _ := riskyOperation()  // Bad!

// No context
return err  // Where did this come from?

// String comparison
if err.Error() == "not found" { }  // Fragile

// Panic for recoverable errors
panic(err)  // Only for truly unrecoverable

// Log and return
log.Println(err)
return err  // Now it's logged twice!

// Capitalized/punctuated messages
errors.New("Failed to connect.")  // Wrong style
```
</anti_patterns>

<summary>
**Error Handling Checklist:**

- [ ] Errors are never ignored (or documented why)
- [ ] Context added with `fmt.Errorf("doing X: %w", err)`
- [ ] Sentinel errors for conditions callers must handle
- [ ] `errors.Is` / `errors.As` for checking (not `==` or string matching)
- [ ] Error strings: lowercase, no punctuation
- [ ] Panic only for unrecoverable programmer errors
- [ ] Either log OR return error, not both
</summary>
