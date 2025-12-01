<overview>
Effective Go defines idiomatic Go code. These patterns make code readable, maintainable, and consistent with the Go ecosystem. Source: https://go.dev/doc/effective_go
</overview>

<formatting>
**Use gofmt for all formatting:**

Go has no style debates. Run `gofmt` (or `gofmt -s` for simplifications):
- Tabs for indentation
- No line length limit (wrap naturally)
- Control structures: braces required, no parentheses

```go
// gofmt handles this automatically
if err != nil {
    return err
}
```
</formatting>

<naming>
**Package names:**
- Short, concise, lowercase
- No underscores or mixedCaps
- Singular (e.g., `user` not `users`)
- Avoid: `utils`, `common`, `base`, `helpers`

**Exported vs unexported:**
- `Uppercase` = exported (public)
- `lowercase` = unexported (private)

**Getters:**
- `obj.Owner()` not `obj.GetOwner()`
- Setter: `obj.SetOwner(value)`

**Interface names:**
- Single-method: method name + "-er" (`Reader`, `Writer`, `Closer`)
- Multi-method: descriptive noun (`ReadWriter`, `ResponseWriter`)

**MixedCaps:**
- Always use `MixedCaps` or `mixedCaps`
- Never underscores (except in tests: `Test_Foo_Bar`)
</naming>

<control_structures>
**If statements:**

```go
// Initialization common
if err := doSomething(); err != nil {
    return err
}

// Avoid else when if ends with return/break/continue
if condition {
    return x
}
return y  // not: else { return y }
```

**For loops:**

```go
// C-style
for i := 0; i < n; i++ { }

// While-style
for condition { }

// Infinite
for { }

// Range
for i, v := range slice { }
for k, v := range map { }
for i, r := range "string" { }  // r is rune
```

**Switch:**

```go
// No fall-through by default (unlike C)
switch value {
case 1:
    // ...
case 2, 3:
    // multiple values
default:
    // ...
}

// Type switch
switch v := x.(type) {
case int:
    // v is int
case string:
    // v is string
}

// Expression-less (like if-else chain)
switch {
case x < 0:
    return -1
case x > 0:
    return 1
default:
    return 0
}
```
</control_structures>

<functions>
**Multiple return values:**

```go
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

**Named returns:**

```go
// Useful for documentation, but avoid naked returns in long functions
func (f *File) Read(b []byte) (n int, err error) {
    // n and err are initialized to zero values
    return  // naked return - use sparingly
}
```

**Defer:**

```go
func process(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()  // runs when function exits

    // work with file...
    return nil
}

// Multiple defers run LIFO (last-in, first-out)
```
</functions>

<data_types>
**Arrays vs Slices:**

| Arrays | Slices |
|--------|--------|
| Fixed size `[N]T` | Dynamic `[]T` |
| Value type (copied) | Reference to underlying array |
| Rarely used directly | Primary collection type |

```go
// Slice creation
s := []int{1, 2, 3}
s := make([]int, len, cap)

// Slice operations
s = append(s, 4, 5)
s2 := s[1:3]  // shares underlying array
```

**Maps:**

```go
m := make(map[string]int)
m["key"] = value

// Check existence
v, ok := m["key"]
if !ok {
    // key doesn't exist
}

// Delete
delete(m, "key")

// Iteration order is random
```

**new vs make:**

| `new(T)` | `make(T)` |
|----------|-----------|
| Allocates zeroed memory | Initializes slices, maps, channels |
| Returns `*T` | Returns `T` (not pointer) |
| Works for any type | Only slice, map, channel |

```go
p := new(int)        // *int, points to 0
s := make([]int, 10) // []int with len 10
m := make(map[string]int)
ch := make(chan int, 5)
```
</data_types>

<interfaces>
**Implicit satisfaction:**

```go
// No "implements" keyword
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Any type with Read method satisfies Reader
type MyReader struct{}
func (r MyReader) Read(p []byte) (int, error) { ... }
```

**Type assertions:**

```go
// Unsafe (panics if wrong type)
s := value.(string)

// Safe (comma-ok idiom)
s, ok := value.(string)
if !ok {
    // not a string
}
```

**Empty interface:**

```go
var x interface{}  // or: var x any
x = 42
x = "hello"
// Use sparingly - loses type safety
```
</interfaces>

<embedding>
**Struct embedding:**

```go
type ReadWriter struct {
    *Reader
    *Writer
}
// ReadWriter has all methods of Reader and Writer

type Server struct {
    *log.Logger  // Server gains Printf, Println, etc.
}
```

**Interface embedding:**

```go
type ReadWriter interface {
    Reader
    Writer
}
```
</embedding>

<concurrency_basics>
**Goroutines:**

```go
go doSomething()  // starts concurrent execution
go func() {
    // anonymous function
}()
```

**Channels:**

```go
ch := make(chan int)     // unbuffered
ch := make(chan int, 10) // buffered

ch <- value  // send
v := <-ch    // receive

close(ch)    // close (sender only)
```

**Select:**

```go
select {
case v := <-ch1:
    // received from ch1
case ch2 <- x:
    // sent to ch2
case <-time.After(time.Second):
    // timeout
default:
    // non-blocking
}
```
</concurrency_basics>

<errors>
**Error handling:**

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}
```

**Custom errors:**

```go
type PathError struct {
    Op   string
    Path string
    Err  error
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}

func (e *PathError) Unwrap() error {
    return e.Err
}
```

**Panic and recover:**

```go
// Only for truly unrecoverable situations
panic("impossible state")

// Recover in deferred function
defer func() {
    if r := recover(); r != nil {
        log.Printf("recovered: %v", r)
    }
}()
```
</errors>

<best_practices>
**Zero values should be useful:**

```go
// Good: zero Buffer is ready to use
var buf bytes.Buffer
buf.WriteString("hello")

// Document if initialization required
type Server struct {
    addr string  // must be set before Start
}
```

**Use blank identifier to discard:**

```go
_, err := io.Copy(dst, src)  // discard bytes written

// Compile-time interface check
var _ io.Reader = (*MyType)(nil)
```

**Implement Stringer for custom formatting:**

```go
func (u User) String() string {
    return fmt.Sprintf("User(%s)", u.Name)
}
```
</best_practices>
