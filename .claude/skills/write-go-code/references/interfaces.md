<overview>
Interfaces are central to good Go design. They enable decoupling, testability, and all SOLID principles. Understanding interface design is essential for idiomatic Go.
</overview>

<basics>
**Interface definition:**

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

**Implicit satisfaction:**

```go
// No "implements" keyword needed
type MyReader struct{}

func (r *MyReader) Read(p []byte) (int, error) {
    // implementation
}

// MyReader automatically satisfies Reader interface
var r Reader = &MyReader{}
```

**Compile-time check:**

```go
// Verify type implements interface at compile time
var _ Reader = (*MyReader)(nil)
```
</basics>

<design_principles>
**Keep interfaces small:**

```go
// Good: single method
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Compose when needed
type ReadWriter interface {
    Reader
    Writer
}
```

**Define where used, not where implemented:**

```go
// Bad: interface defined with implementation
package postgres

type UserStore interface {  // defined here...
    Get(id string) (*User, error)
    Save(u *User) error
}

type Store struct { db *sql.DB }
func (s *Store) Get(id string) (*User, error) { ... }
func (s *Store) Save(u *User) error { ... }

// Good: interface defined where consumed
package service

// Interface defined by the consumer
type UserStore interface {
    Get(id string) (*User, error)
    Save(u *User) error
}

type UserService struct {
    store UserStore  // accepts any implementation
}
```

**Accept interfaces, return structs:**

```go
// Good: accepts interface, returns concrete type
func NewServer(logger Logger) *Server {
    return &Server{logger: logger}
}

// Caller can pass any Logger implementation
srv := NewServer(zapLogger)
srv := NewServer(testLogger)
```
</design_principles>

<common_patterns>
**Single-method interfaces:**

```go
// Standard library examples
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Closer interface { Close() error }
type Stringer interface { String() string }

// Custom examples
type Validator interface { Validate() error }
type Processor interface { Process(data []byte) error }
type Handler interface { Handle(ctx context.Context, req Request) Response }
```

**Functional interfaces:**

```go
// Define function types for single-method interfaces
type HandlerFunc func(ctx context.Context, req Request) Response

func (f HandlerFunc) Handle(ctx context.Context, req Request) Response {
    return f(ctx, req)
}

// Usage: pass either struct or function
Process(myHandler)
Process(HandlerFunc(func(ctx context.Context, req Request) Response {
    return Response{}
}))
```

**Optional methods pattern:**

```go
// Base interface
type Store interface {
    Get(id string) ([]byte, error)
}

// Optional capability
type Transactional interface {
    BeginTx() (Tx, error)
}

// Check at runtime
func process(store Store) error {
    if tx, ok := store.(Transactional); ok {
        // use transactions
        t, _ := tx.BeginTx()
        defer t.Rollback()
        // ...
    }
    // fallback to non-transactional
}
```
</common_patterns>

<type_assertions>
**Type assertion (unsafe):**

```go
s := value.(string)  // panics if not string
```

**Type assertion (safe):**

```go
s, ok := value.(string)
if !ok {
    // not a string
}
```

**Type switch:**

```go
switch v := value.(type) {
case string:
    fmt.Println("string:", v)
case int:
    fmt.Println("int:", v)
case error:
    fmt.Println("error:", v.Error())
default:
    fmt.Printf("unknown type: %T\n", v)
}
```
</type_assertions>

<empty_interface>
**The empty interface (`any`):**

```go
var x any  // same as interface{}
x = 42
x = "hello"
x = []int{1, 2, 3}
```

**When to use:**
- Truly generic containers (before generics)
- JSON unmarshaling to unknown structures
- Logging/debugging utilities

**When NOT to use:**
- When you know the types (use generics or specific interfaces)
- To avoid thinking about types (defeats type safety)

```go
// Bad: any for known types
func Process(data any) { ... }

// Good: specific interface
func Process(data Processable) { ... }

// Good: generics (Go 1.18+)
func Process[T Processable](data T) { ... }
```
</empty_interface>

<testing>
**Interfaces enable testing:**

```go
// Production code
type UserService struct {
    store UserStore  // interface
}

// Test with fake
type fakeStore struct {
    users map[string]*User
    err   error
}

func (f *fakeStore) Get(id string) (*User, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.users[id], nil
}

func TestUserService_Get(t *testing.T) {
    store := &fakeStore{
        users: map[string]*User{"1": {ID: "1", Name: "Alice"}},
    }
    svc := &UserService{store: store}

    user, err := svc.Get("1")
    // assertions...
}
```
</testing>

<anti_patterns>
**Large interfaces:**

```go
// Bad: too many methods
type Repository interface {
    Get(id string) (*Entity, error)
    List() ([]*Entity, error)
    Create(e *Entity) error
    Update(e *Entity) error
    Delete(id string) error
    Search(q string) ([]*Entity, error)
    Count() (int, error)
    Exists(id string) (bool, error)
}

// Good: split by use case
type Getter interface { Get(id string) (*Entity, error) }
type Lister interface { List() ([]*Entity, error) }
type Writer interface { Create(*Entity) error; Update(*Entity) error }
```

**Interface for one implementation:**

```go
// Bad: unnecessary abstraction
type UserService interface {
    GetUser(id string) (*User, error)
}

type userService struct { ... }

// Good: just use the concrete type
type UserService struct { ... }
```

**Pointer to interface:**

```go
// Almost always wrong
func Process(r *io.Reader) { }  // Bad

// Correct
func Process(r io.Reader) { }   // Good
```

**Returning interfaces:**

```go
// Bad: hides implementation details
func NewReader() Reader { return &myReader{} }

// Good: return concrete type
func NewReader() *MyReader { return &MyReader{} }
```
</anti_patterns>

<guidelines>
**Interface Design Guidelines:**

1. **Start with no interfaces** - add when you need abstraction
2. **Keep small** - 1-2 methods ideal, max 5
3. **Define at consumer** - where it's used, not implemented
4. **Name by behavior** - Reader, Writer, Handler (not IReader)
5. **Accept interfaces** - parameters should be interfaces
6. **Return structs** - return values should be concrete types
7. **No interface for single implementation** - premature abstraction
</guidelines>

<summary>
**Quick Reference:**

| Pattern | When to Use |
|---------|-------------|
| Single-method interface | Maximum flexibility |
| Interface composition | Combining behaviors |
| Define at consumer | Always (not at implementation) |
| Type assertion | Runtime type checking |
| Type switch | Handling multiple types |
| Empty interface | Truly unknown types only |
| Functional interface | When behavior is a single function |
</summary>
