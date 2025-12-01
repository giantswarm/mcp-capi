<overview>
Go naming conventions are not just style - they convey meaning. Names indicate scope, visibility, and intent. Follow these conventions for idiomatic Go.
</overview>

<packages>
**Package Names:**

| Rule | Example |
|------|---------|
| Lowercase only | `user`, `http`, `json` |
| Short, single word | `sync` not `synchronization` |
| Singular | `user` not `users` |
| No underscores | `httputil` not `http_util` |
| No mixedCaps | `strconv` not `strConv` |

**Avoid these package names:**
- `utils`, `util` - too vague
- `common`, `shared` - becomes a dumping ground
- `helpers`, `base` - unclear purpose
- `misc`, `general` - undefined scope

**Package name is part of the identifier:**

```go
// Bad: stutters
package user
type UserService struct{}  // user.UserService

// Good: no stutter
package user
type Service struct{}      // user.Service
```
</packages>

<exported_names>
**Exported (Public) Names:**

Use `MixedCaps` for multi-word names:

```go
type HttpClient struct{}   // Bad
type HTTPClient struct{}   // Good (acronym)

type JsonParser struct{}   // Bad
type JSONParser struct{}   // Good

type UrlBuilder struct{}   // Bad
type URLBuilder struct{}   // Good
```

**Common acronyms (keep uppercase):**
- HTTP, URL, API, ID, JSON, XML, HTML, SQL, SSH, TCP, UDP, TLS, RPC

**Type names:**
- Noun or noun phrase
- Describes what it is, not what it does
- Examples: `Reader`, `Config`, `Response`, `User`

**Function/Method names:**
- Verb or verb phrase (usually)
- Describes what it does
- Examples: `Read`, `Parse`, `NewServer`, `String`
</exported_names>

<unexported_names>
**Unexported (Private) Names:**

Same rules, just start with lowercase:

```go
type request struct{}          // private type
func processItems() {}         // private function
var defaultTimeout time.Duration  // private variable
```
</unexported_names>

<variables>
**Variable Names:**

| Scope | Style | Example |
|-------|-------|---------|
| Package-level | descriptive | `maxRetries`, `defaultTimeout` |
| Function-level | shorter | `users`, `config`, `result` |
| Loop/short block | very short | `i`, `j`, `k`, `v`, `r`, `w` |

**Common short names:**

| Name | Meaning |
|------|---------|
| `i`, `j`, `k` | loop indices |
| `n` | count, size |
| `v` | value (in range) |
| `k` | key (in range) |
| `r` | reader |
| `w` | writer |
| `b` | byte slice, buffer |
| `s` | string |
| `err` | error |
| `ctx` | context.Context |
| `t` | *testing.T |
| `ok` | boolean result |

**Receivers:**

```go
// Use 1-2 letter abbreviation of type name
func (s *Server) Start() {}
func (c *Client) Do() {}
func (u *User) Validate() {}
func (req *Request) Send() {}
```
</variables>

<getters_setters>
**Getters and Setters:**

```go
// Getter: NO "Get" prefix
func (u *User) Name() string { return u.name }

// Setter: "Set" prefix
func (u *User) SetName(name string) { u.name = name }

// Not:
func (u *User) GetName() string { ... }  // Wrong!
```
</getters_setters>

<interfaces>
**Interface Names:**

| Methods | Naming | Example |
|---------|--------|---------|
| Single method | Method + "er" | `Reader`, `Writer`, `Closer` |
| Multiple methods | Descriptive noun | `ReadWriter`, `FileInfo` |

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Stringer interface {
    String() string
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**When -er doesn't work:**

```go
// Process -> Processor (works)
// Execute -> Executor (works)
// Validate -> Validator (works)
// Clean -> Cleaner (works)

// But sometimes:
type Handler interface {  // not "Handler" from Handle
    ServeHTTP(w ResponseWriter, r *Request)
}
```
</interfaces>

<constants>
**Constants:**

```go
// Exported: MixedCaps
const MaxRetries = 3
const DefaultTimeout = 30 * time.Second

// Unexported: mixedCaps
const maxBufferSize = 4096

// Iota for enums
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusComplete
)
```
</constants>

<errors>
**Error Names:**

```go
// Sentinel errors: Err prefix
var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid")

// Error types: Error suffix
type ValidationError struct { ... }
type TimeoutError struct { ... }
```
</errors>

<anti_patterns>
**Avoid:**

```go
// Underscores
user_service    // Bad
userService     // Good

// Redundant type in name
type UserStruct struct{}  // Bad
type User struct{}        // Good

// "I" prefix for interfaces (Java style)
type IReader interface{}  // Bad
type Reader interface{}   // Good

// Long names for short scopes
for index := 0; index < len(items); index++ {}  // Verbose
for i := 0; i < len(items); i++ {}              // Idiomatic

// Abbreviations without context
func Proc(d []byte) {}   // Unclear
func Process(data []byte) {}  // Clear
```
</anti_patterns>

<summary>
**Quick Reference:**

| Item | Convention | Example |
|------|------------|---------|
| Package | lowercase, short | `user`, `http` |
| Exported type | MixedCaps noun | `UserStore`, `HTTPClient` |
| Unexported type | mixedCaps noun | `requestHandler` |
| Interface (1 method) | Method + "er" | `Reader`, `Closer` |
| Function | MixedCaps verb | `Parse`, `NewServer` |
| Getter | No "Get" | `user.Name()` |
| Setter | "Set" prefix | `user.SetName(n)` |
| Receiver | 1-2 letters | `(s *Server)` |
| Local var (short scope) | 1-3 chars | `i`, `err`, `ctx` |
| Constant | MixedCaps | `MaxRetries` |
| Error var | "Err" prefix | `ErrNotFound` |
| Error type | "Error" suffix | `ValidationError` |
</summary>
