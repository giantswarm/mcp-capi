<overview>
Go project structure should be appropriate to project size. Don't over-engineer small projects; don't under-structure large ones.
</overview>

<size_guide>
**Structure by Project Size:**

| Size | Structure |
|------|-----------|
| Small (< 5 files) | Flat, single directory |
| Medium (5-20 files) | `cmd/`, `internal/` |
| Large (20+ files, multiple binaries) | Full layout |
| Library (for import by others) | Minimal, focused on public API |
</size_guide>

<small_projects>
**Small Projects (flat structure):**

```
myproject/
├── go.mod
├── main.go
├── config.go
├── handler.go
└── handler_test.go
```

No subdirectories needed. Everything in the root package.
</small_projects>

<medium_projects>
**Medium Projects:**

```
myproject/
├── go.mod
├── main.go                 # or cmd/myapp/main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── handler.go
│   │   └── handler_test.go
│   └── store/
│       ├── store.go
│       └── postgres.go
└── README.md
```
</medium_projects>

<large_projects>
**Large Projects:**

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
│   │   ├── user/
│   │   └── order/
│   ├── handler/
│   │   ├── http/
│   │   └── grpc/
│   └── store/
│       ├── postgres/
│       └── redis/
├── pkg/                    # public packages (optional)
│   └── client/
├── api/                    # API definitions (OpenAPI, protobuf)
├── scripts/                # build/deploy scripts
└── README.md
```
</large_projects>

<key_directories>
**Key Directories:**

<directory name="cmd">
Application entry points. Each subdirectory is a binary.

```
cmd/
├── server/
│   └── main.go    # builds to "server" binary
└── worker/
    └── main.go    # builds to "worker" binary
```

`main.go` should be thin - just wiring:
```go
func main() {
    cfg := config.Load()
    db := database.Connect(cfg.DB)
    svc := service.New(db)
    server := http.NewServer(svc)
    log.Fatal(server.ListenAndServe(cfg.Port))
}
```
</directory>

<directory name="internal">
Private packages. Go compiler enforces that `internal/` packages can only be imported within the same module.

Use for:
- Implementation details
- Code that shouldn't be imported by other projects
- Most of your application code

```
internal/
├── config/     # configuration loading
├── domain/     # business logic
├── handler/    # HTTP/gRPC handlers
└── store/      # data persistence
```
</directory>

<directory name="pkg">
Public packages intended for external use. Optional - use only if you have code meant to be imported by other projects.

```
pkg/
└── client/     # SDK for your API
```

**Note:** If you're not building a library, you probably don't need `pkg/`.
</directory>
</key_directories>

<organization>
**Organize by Domain, Not Layer:**

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
│   ├── user.go          # types
│   ├── service.go       # business logic
│   ├── store.go         # persistence
│   └── handler.go       # HTTP
├── order/
│   ├── order.go
│   ├── service.go
│   └── handler.go
└── notification/
    └── sender.go
```

**Benefits:**
- Related code together
- Clear ownership
- Easy to extract as library
- Reduces cross-package imports
</organization>

<dependencies>
**Dependency Direction:**

```
main (highest level)
  ↓
handlers (HTTP, CLI)
  ↓
services (business logic)
  ↓
domain (types, interfaces)
```

Lower-level packages should NOT import higher-level packages.

**Define interfaces at the consumer:**

```go
// In handler package (consumer)
type UserService interface {
    Get(id string) (*user.User, error)
}

type Handler struct {
    users UserService
}

// In main.go (wiring)
userSvc := user.NewService(db)
handler := handler.New(userSvc)  // passes concrete type
```
</dependencies>

<naming>
**Package Naming:**

Good:
- `user`, `order`, `http`, `postgres`
- Short, lowercase, single word
- Describes what it provides

Bad:
- `utils`, `common`, `helpers` (too vague)
- `userService`, `orderHandler` (redundant with parent dir)
- `models` (what kind of models?)

**Avoid stuttering:**

```go
// Bad
package user
type UserService struct{}  // user.UserService

// Good
package user
type Service struct{}      // user.Service
```
</naming>

<module_init>
**Module Initialization:**

```bash
# Initialize module
go mod init github.com/yourorg/yourproject

# Add dependencies
go mod tidy

# Verify
go mod verify
```

Module path should match where code is hosted:
- `github.com/org/repo`
- `yourcompany.com/internal/project`
</module_init>

<patterns>
**Common Patterns:**

**Single binary with internal packages:**
```
myapp/
├── go.mod
├── main.go
└── internal/
    ├── config/
    ├── server/
    └── store/
```

**Library for others to import:**
```
mylib/
├── go.mod
├── mylib.go         # public API
├── options.go       # public options
└── internal/        # private implementation
    └── impl/
```

**Monorepo with multiple services:**
```
platform/
├── go.mod           # single module
├── cmd/
│   ├── api-server/
│   ├── worker/
│   └── cli/
├── internal/
│   ├── api/
│   ├── worker/
│   └── shared/
└── pkg/             # shared client libraries
    └── client/
```
</patterns>

<avoid>
**Avoid:**

- `src/` directory (Go convention doesn't use it)
- `vendor/` checked in (usually unnecessary with go modules)
- Deep nesting (`internal/pkg/util/helpers/strings/`)
- Import cycles (Go compiler prevents, but design to avoid)
- `pkg/` without public packages
- `utils`, `common`, `base`, `helpers` packages
- Organizing by layer instead of domain
</avoid>

<guidelines>
**Guidelines:**

1. **Start simple** - flat structure, add directories as needed
2. **Use `internal/`** - for most application code
3. **Keep `main` thin** - only wiring, no business logic
4. **Domain packages** - group by feature, not technical layer
5. **Dependencies flow down** - high-level imports low-level
6. **Interfaces at consumer** - define where used
7. **Match project size** - don't over-engineer
</guidelines>
