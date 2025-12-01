<overview>
Anti-patterns are common practices that seem reasonable but lead to problems. Recognizing them helps you write better Go code and review others' code effectively.
</overview>

<critical>
**Critical Anti-Patterns (fix immediately):**

<pattern name="ignoring-errors">
**Ignoring Errors**

```go
// Bad: silent failure
result, _ := doSomething()

// Bad: checking but not using
if err := doSomething(); err != nil {
    // empty block or just log
}

// Good: handle or return
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// If truly safe to ignore, document why:
_ = conn.Close() // best-effort cleanup, already handled response
```
</pattern>

<pattern name="goroutine-leaks">
**Goroutine Leaks**

```go
// Bad: goroutine can never exit
go func() {
    for item := range ch {  // blocked forever if ch never closes
        process(item)
    }
}()

// Good: provide exit mechanism
go func() {
    for {
        select {
        case <-ctx.Done():
            return  // clean exit
        case item, ok := <-ch:
            if !ok {
                return  // channel closed
            }
            process(item)
        }
    }
}()
```
</pattern>

<pattern name="data-races">
**Data Races**

```go
// Bad: concurrent access without synchronization
var counter int
go func() { counter++ }()
go func() { counter++ }()

// Good: use mutex
var (
    mu      sync.Mutex
    counter int
)
go func() {
    mu.Lock()
    counter++
    mu.Unlock()
}()

// Better: use atomic or channel
var counter atomic.Int64
go func() { counter.Add(1) }()
```

Run `make test` to detect races.
</pattern>

<pattern name="panic-for-errors">
**Using Panic for Recoverable Errors**

```go
// Bad: crashes for normal error conditions
func Parse(s string) int {
    n, err := strconv.Atoi(s)
    if err != nil {
        panic(err)
    }
    return n
}

// Good: return error
func Parse(s string) (int, error) {
    return strconv.Atoi(s)
}
```

Reserve panic for truly unrecoverable situations (programmer error, impossible state).
</pattern>
</critical>

<important>
**Important Anti-Patterns (should fix):**

<pattern name="large-interfaces">
**Large Interfaces**

```go
// Bad: forces unnecessary dependencies
type Repository interface {
    Get(id string) (*Entity, error)
    List() ([]*Entity, error)
    Create(e *Entity) error
    Update(e *Entity) error
    Delete(id string) error
    Search(q string) ([]*Entity, error)
    Count() (int, error)
}

// Good: small, focused interfaces
type Getter interface { Get(id string) (*Entity, error) }
type Lister interface { List() ([]*Entity, error) }

// Compose only when needed
type ReadWriter interface {
    Getter
    Lister
}
```
</pattern>

<pattern name="init-side-effects">
**init() with Side Effects**

```go
// Bad: hidden side effects, hard to test
func init() {
    db, _ = sql.Open("postgres", os.Getenv("DB_URL"))
    http.HandleFunc("/", handler)
}

// Good: explicit initialization
func main() {
    db, err := sql.Open("postgres", os.Getenv("DB_URL"))
    if err != nil {
        log.Fatal(err)
    }
    http.HandleFunc("/", NewHandler(db))
}
```

`init()` is acceptable for: registering drivers, compile-time checks, simple variable initialization.
</pattern>

<pattern name="god-packages">
**God Packages (utils, common, helpers)**

```go
// Bad: dumping ground
package utils

func FormatDate(t time.Time) string { ... }
func ValidateEmail(s string) bool { ... }
func HashPassword(s string) string { ... }
func RetryHTTP(f func()) error { ... }

// Good: domain-specific packages
package user
func ValidateEmail(s string) bool { ... }
func HashPassword(s string) string { ... }

package http
func Retry(f func()) error { ... }
```
</pattern>

<pattern name="concrete-dependencies">
**Depending on Concrete Types**

```go
// Bad: tight coupling
type Handler struct {
    store *postgres.Store  // depends on specific implementation
}

// Good: depend on interface
type UserStore interface {
    Get(id string) (*User, error)
}

type Handler struct {
    store UserStore  // accepts any implementation
}
```
</pattern>

<pattern name="error-string-checking">
**Checking Error Strings**

```go
// Bad: fragile, breaks with wrapped errors
if err.Error() == "not found" { ... }
if strings.Contains(err.Error(), "timeout") { ... }

// Good: use errors.Is/As
if errors.Is(err, ErrNotFound) { ... }

var netErr net.Error
if errors.As(err, &netErr) && netErr.Timeout() { ... }
```
</pattern>

<pattern name="factory-does-too-much">
**Factory Functions That Do Too Much**

```go
// Bad: creates AND connects (tight coupling)
func NewClient() *Client {
    c := &Client{}
    c.Connect()  // caller can't control when this happens
    return c
}

// Good: factory only creates
func NewClient(config Config) *Client {
    return &Client{config: config}
}

// Caller controls connection
client := NewClient(cfg)
if err := client.Connect(ctx); err != nil {
    // handle
}
```
</pattern>
</important>

<stylistic>
**Stylistic Anti-Patterns (improve readability):**

<pattern name="stuttering">
**Name Stuttering**

```go
// Bad: package.Type repeats
package user
type UserService struct{}  // user.UserService

// Good: no repetition
package user
type Service struct{}      // user.Service
```
</pattern>

<pattern name="get-prefix">
**Get Prefix on Getters**

```go
// Bad: Java style
func (u *User) GetName() string { return u.name }

// Good: Go style
func (u *User) Name() string { return u.name }
```
</pattern>

<pattern name="unnecessary-else">
**Unnecessary Else**

```go
// Bad: unnecessary else
if err != nil {
    return err
} else {
    return nil
}

// Good: early return
if err != nil {
    return err
}
return nil
```
</pattern>

<pattern name="naked-returns">
**Naked Returns in Long Functions**

```go
// Bad: unclear what's returned in long function
func process(data []byte) (result string, err error) {
    // 50 lines of code...
    return  // what values?
}

// Good: explicit returns
func process(data []byte) (string, error) {
    // 50 lines of code...
    return result, nil
}
```

Naked returns are okay in short functions (< 10 lines) where named returns add clarity.
</pattern>

<pattern name="pointer-to-interface">
**Pointer to Interface**

```go
// Bad: almost always wrong
func Process(r *io.Reader) error { ... }

// Good: interfaces are already references
func Process(r io.Reader) error { ... }
```
</pattern>

<pattern name="unnecessary-break">
**Break in Switch**

```go
// Bad: unnecessary (Go doesn't fall through)
switch v {
case 1:
    doOne()
    break  // unnecessary
case 2:
    doTwo()
    break  // unnecessary
}

// Good: implicit break
switch v {
case 1:
    doOne()
case 2:
    doTwo()
}
```
</pattern>

<pattern name="excessive-comments">
**Comments That Restate Code**

```go
// Bad: obvious comment
// increment i by 1
i++

// set name to "John"
name = "John"

// Good: comment explains WHY
// Skip the first element as it's always the header row
for i := 1; i < len(rows); i++ { ... }
```
</pattern>
</stylistic>

<concurrency_antipatterns>
**Concurrency Anti-Patterns:**

<pattern name="sleep-synchronization">
**Using Sleep for Synchronization**

```go
// Bad: timing-dependent, flaky
go doWork()
time.Sleep(100 * time.Millisecond)  // hope it's done
checkResult()

// Good: proper synchronization
done := make(chan struct{})
go func() {
    doWork()
    close(done)
}()
<-done
checkResult()
```
</pattern>

<pattern name="closing-from-receiver">
**Closing Channel from Receiver**

```go
// Bad: sender might panic
go func() {
    for v := range ch {
        if v == 0 {
            close(ch)  // sender will panic!
        }
    }
}()

// Good: sender closes
go func() {
    defer close(ch)
    for _, v := range values {
        ch <- v
    }
}()
```

Rule: only the sender should close a channel.
</pattern>

<pattern name="unbounded-goroutines">
**Unbounded Goroutines**

```go
// Bad: can spawn millions of goroutines
for _, item := range items {
    go process(item)
}

// Good: worker pool
sem := make(chan struct{}, maxWorkers)
for _, item := range items {
    sem <- struct{}{}
    go func(item Item) {
        defer func() { <-sem }()
        process(item)
    }(item)
}
```
</pattern>
</concurrency_antipatterns>

<detection>
**Tools to Detect Anti-Patterns:**

```bash
# Linting (includes vet, static analysis)
make lint

# Tests with race detection
make test

# Find unchecked errors
errcheck ./...

# Security issues
gosec ./...
```
</detection>

<summary>
**Quick Reference - Common Anti-Patterns:**

| Anti-Pattern | Problem | Solution |
|--------------|---------|----------|
| Ignored errors | Silent failures | Handle or return with context |
| Goroutine leaks | Resource exhaustion | Provide exit mechanism (context) |
| Data races | Undefined behavior | Mutex, atomic, or channels |
| Panic for errors | Crashes program | Return error values |
| Large interfaces | Tight coupling | Small interfaces (1-2 methods) |
| God packages | Unclear ownership | Domain-specific packages |
| Error string checking | Fragile | errors.Is/As |
| Name stuttering | Redundancy | Let package name provide context |
</summary>
