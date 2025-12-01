# Workflow: Structure a Go Project

<required_reading>
**Read these reference files NOW:**
1. references/project-structure.md
2. references/solid-principles.md
</required_reading>

<process>

<step name="assess-project-size">
**Determine appropriate structure:**

| Project Size | Recommended Structure |
|--------------|----------------------|
| Small (single binary, few packages) | Flat or minimal structure |
| Medium (multiple packages, internal code) | Standard layout with `cmd/`, `internal/` |
| Large (multiple binaries, public packages) | Full layout with `pkg/`, `internal/`, `cmd/` |
| Library (imported by others) | Focus on public API, minimal directories |
</step>

<step name="create-base-structure">
**Set up directory structure:**

**Minimal (small projects):**
```
myproject/
├── go.mod
├── main.go
├── config.go
└── handler.go
```

**Standard (medium projects):**
```
myproject/
├── go.mod
├── main.go                 # or cmd/myapp/main.go
├── internal/
│   ├── config/
│   ├── handler/
│   └── storage/
└── README.md
```

**Full (large projects):**
```
myproject/
├── go.mod
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── cli/
│       └── main.go
├── internal/               # private packages
│   ├── config/
│   ├── domain/
│   ├── handler/
│   └── storage/
├── pkg/                    # public packages (optional)
│   └── client/
└── README.md
```
</step>

<step name="understand-key-directories">
**Know what goes where:**

| Directory | Purpose | Importable? |
|-----------|---------|-------------|
| `cmd/` | Application entry points (`main.go`) | No |
| `internal/` | Private application code | Only within module |
| `pkg/` | Public packages for external use | Yes, by anyone |
| Root | Library code or small project | Yes |

**`internal/` is special:** Go enforces that `internal/` packages can only be imported by code in the same module. Use it for implementation details.
</step>

<step name="design-packages">
**Organize by domain, not layer:**

```
# Bad: organized by technical layer
internal/
├── models/
├── handlers/
├── services/
└── repositories/

# Good: organized by domain
internal/
├── user/
│   ├── user.go          # domain types
│   ├── store.go         # persistence
│   └── handler.go       # HTTP handlers
├── order/
│   ├── order.go
│   ├── store.go
│   └── handler.go
└── email/
    └── sender.go
```

Benefits:
- Related code is together
- Dependencies are clear
- Easier to understand scope
- Packages can be extracted as libraries
</step>

<step name="keep-main-thin">
**main.go should only wire things together:**

```go
// cmd/server/main.go
package main

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Create dependencies
    db := database.Connect(cfg.DatabaseURL)
    userStore := user.NewStore(db)
    userHandler := user.NewHandler(userStore)

    // Wire up and run
    server := http.NewServer(userHandler)
    log.Fatal(server.ListenAndServe(cfg.Port))
}
```

Push all business logic to packages. `main` only does:
- Configuration loading
- Dependency creation
- Wiring components together
- Starting/stopping the application
</step>

<step name="manage-dependencies">
**Follow Dependency Inversion:**

Define interfaces where they're used:

```go
// internal/user/handler.go
package user

// Store defines what the handler needs from persistence.
// Defined here, not in the storage package.
type Store interface {
    Get(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, u *User) error
}

type Handler struct {
    store Store
}
```

Concrete implementations are created in `main` and injected:

```go
// cmd/server/main.go
store := postgres.NewUserStore(db)  // implements user.Store
handler := user.NewHandler(store)
```
</step>

<step name="avoid-import-cycles">
**Structure to prevent cycles:**

If package A imports B and B needs something from A:
1. Extract shared types to a third package
2. Define interface in the consumer
3. Pass the dependency via constructor

```
# Problem: cycle between user and order
user → order (user needs order history)
order → user (order needs user details)

# Solution: extract shared types or use interfaces
user → order (user.Service accepts OrderGetter interface)
order (implements user.OrderGetter)
```
</step>

<step name="initialize-module">
**Set up Go modules:**

```bash
# Initialize module
go mod init github.com/yourorg/yourproject

# Add dependencies as you import them
go mod tidy

# Verify module
go mod verify
```

Module path conventions:
- GitHub projects: `github.com/org/repo`
- Internal projects: `yourcompany.com/project`
- Unique path that matches where code is hosted
</step>

</process>

<anti_patterns>
**Avoid:**
- `utils`, `common`, `helpers`, `misc` packages (too generic)
- Organizing by technical layer (`models/`, `services/`, `repositories/`)
- Deep directory nesting (keep it shallow)
- Circular imports (design flaw)
- `pkg/` for non-public code (use `internal/`)
- Giant packages with many responsibilities
- Business logic in `main.go`
</anti_patterns>

<success_criteria>
Project structure is good when:
- [ ] Structure matches project size (not over-engineered)
- [ ] Packages organized by domain/feature
- [ ] `main.go` is thin (only wiring)
- [ ] `internal/` used for private code
- [ ] No import cycles
- [ ] Dependencies flow inward (DIP)
- [ ] Package names are clear and specific
- [ ] Easy to understand where new code belongs
</success_criteria>
