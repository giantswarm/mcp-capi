# Workflow: Write New Go Code

<required_reading>
**Read these reference files NOW before writing code:**
1. references/effective-go.md
2. references/naming-conventions.md
3. references/interfaces.md
</required_reading>

<process>

<step name="understand-requirements">
**Clarify what you're building:**
- What is the single responsibility of this code?
- What inputs does it accept? What outputs does it produce?
- What errors can occur?
- Who are the consumers (internal package, external API, CLI)?
</step>

<step name="design-interfaces-first">
**Define behavior before implementation:**

If this code needs to be testable or extensible, define interfaces:

```go
// Describe the behavior you need
type UserStore interface {
    Get(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

Apply Interface Segregation: keep interfaces small. A function that only reads shouldn't depend on a write interface.
</step>

<step name="choose-appropriate-types">
**Select the right data structures:**

| Need | Use |
|------|-----|
| Fixed collection | Array `[N]T` |
| Dynamic collection | Slice `[]T` |
| Key-value lookup | Map `map[K]V` |
| Optional value | Pointer `*T` or comma-ok pattern |
| Concurrent access | Channel or sync primitives |
| Multiple related values | Struct |

Consider zero values: design types so the zero value is useful.

```go
// Good: zero value is valid (empty buffer)
type Buffer struct {
    data []byte
}

// Requires initialization - document this
type Server struct {
    addr string  // must be set
}
```
</step>

<step name="implement-with-good-names">
**Write the implementation:**

Follow naming conventions:
- Package name: short, lowercase, no underscores
- Exported names: describe what they return or do
- Local variables: short for short scope
- Receivers: one or two letter abbreviation of type

```go
package user

// User represents a registered user in the system.
type User struct {
    ID    string
    Name  string
    Email string
}

// Store provides persistence for users.
type Store struct {
    db Database
}

// Get retrieves a user by ID. Returns ErrNotFound if the user doesn't exist.
func (s *Store) Get(ctx context.Context, id string) (*User, error) {
    u, err := s.db.Query(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("querying user %s: %w", id, err)
    }
    return u, nil
}
```
</step>

<step name="handle-errors-properly">
**Add appropriate error handling:**

- Return errors, don't panic (unless truly unrecoverable)
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Define sentinel errors for expected conditions callers must handle
- Document which errors a function can return

```go
var (
    ErrNotFound = errors.New("user not found")
    ErrInvalid  = errors.New("invalid user data")
)
```
</step>

<step name="document-exported-items">
**Add doc comments to exported items:**

```go
// Package user provides user management functionality.
package user

// User represents a registered user in the system.
// The zero value is not valid; use New to create users.
type User struct { ... }

// New creates a User with the given name and email.
// It returns ErrInvalid if name or email is empty.
func New(name, email string) (*User, error) { ... }
```

Doc comments are complete sentences starting with the name being documented.
</step>

<step name="verify">
**Run verification:**

```bash
gofmt -s -w .
make lint
make build
make test
```

Fix any issues before considering the code complete.
</step>

</process>

<anti_patterns>
**Avoid:**
- Returning unexported types from exported functions
- Giant interfaces (keep them small, ideally 1-2 methods)
- `init()` functions that do heavy work or can fail
- Package names matching standard library (`http`, `json`, etc.)
- `utils`, `common`, `misc` package names
- Getters named `GetX()` - use `X()` instead
- Ignoring errors with `_`
- Naked returns in long functions
</anti_patterns>

<success_criteria>
Code is well-written when:
- [ ] Single responsibility is clear from package/type name
- [ ] Interfaces are small and focused
- [ ] Zero values are useful or initialization is documented
- [ ] Errors are handled and wrapped with context
- [ ] Exported items have doc comments
- [ ] Names follow Go conventions
- [ ] `gofmt`, `make lint`, and `make test` pass
</success_criteria>
