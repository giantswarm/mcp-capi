# Workflow: Refactor Go Code

<required_reading>
**Read these reference files NOW before refactoring:**
1. references/solid-principles.md
2. references/interfaces.md
3. references/anti-patterns.md
</required_reading>

<process>

<step name="identify-problems">
**Identify what needs improvement:**

Common refactoring targets:
- Functions doing too many things (violates SRP)
- Large interfaces that force unnecessary dependencies (violates ISP)
- Concrete type dependencies that prevent testing (violates DIP)
- Duplicated code across packages
- Poor error handling (swallowed errors, missing context)
- Unclear naming
- Tight coupling between packages
</step>

<step name="apply-srp">
**Single Responsibility Principle:**

If a type or function has multiple reasons to change, split it:

```go
// Before: does validation AND persistence
func (s *Service) CreateUser(name, email string) error {
    if name == "" || email == "" {
        return errors.New("invalid input")
    }
    return s.db.Insert(...)
}

// After: separate concerns
func ValidateUser(name, email string) error {
    if name == "" {
        return errors.New("name required")
    }
    if email == "" {
        return errors.New("email required")
    }
    return nil
}

func (s *Service) CreateUser(u *User) error {
    return s.db.Insert(u)
}
```
</step>

<step name="extract-interfaces">
**Interface Segregation Principle:**

Extract interfaces from concrete dependencies:

```go
// Before: depends on concrete type
type Handler struct {
    db *sql.DB  // tight coupling
}

// After: depends on interface (what we actually need)
type UserGetter interface {
    GetUser(ctx context.Context, id string) (*User, error)
}

type Handler struct {
    users UserGetter  // loose coupling, testable
}
```

Keep interfaces small. If a function only reads, it shouldn't depend on a write interface.
</step>

<step name="invert-dependencies">
**Dependency Inversion Principle:**

Move interface definitions to where they're used (consumer side):

```go
// In the consumer package (not the provider):
type Logger interface {
    Info(msg string, args ...any)
}

type Service struct {
    log Logger  // accepts any logger implementation
}
```

Push concrete implementations to `main` or high-level packages.
</step>

<step name="reduce-coupling">
**Reduce package coupling:**

- Move shared types to a separate package if multiple packages need them
- Use interfaces at package boundaries
- Avoid import cycles (Go enforces this, but design for it)
- Consider: would this package make sense as a standalone library?

```go
// Instead of importing a full package for one type:
type Processor interface {
    Process(data []byte) error
}
// Now you don't need to import the concrete implementation
```
</step>

<step name="simplify">
**Simplify aggressively:**

- Remove dead code
- Inline single-use helper functions
- Replace complex conditionals with early returns
- Use table-driven logic instead of long switch statements
- Remove unnecessary abstractions

```go
// Before: nested conditionals
func process(x int) string {
    if x > 0 {
        if x > 10 {
            return "large"
        } else {
            return "small"
        }
    } else {
        return "zero or negative"
    }
}

// After: early returns
func process(x int) string {
    if x <= 0 {
        return "zero or negative"
    }
    if x > 10 {
        return "large"
    }
    return "small"
}
```
</step>

<step name="verify-behavior">
**Ensure behavior is preserved:**

```bash
# Run tests
make test

# Check coverage for regressions
make test-coverage
```

Refactoring should not change behavior. If tests fail, the refactoring introduced a bug.
</step>

</process>

<anti_patterns>
**Avoid during refactoring:**
- Changing behavior while refactoring (do one at a time)
- Creating abstractions for single implementations
- Over-engineering: don't add flexibility you don't need
- Breaking public APIs without versioning
- Refactoring without tests (add tests first if needed)
</anti_patterns>

<success_criteria>
Refactoring is successful when:
- [ ] Each type/function has a single responsibility
- [ ] Interfaces are small (1-2 methods ideal)
- [ ] Dependencies are on interfaces, not concrete types
- [ ] No import cycles
- [ ] All tests pass
- [ ] Code is easier to understand and modify
- [ ] No behavior changes (unless intentional)
</success_criteria>
