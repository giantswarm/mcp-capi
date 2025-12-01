<overview>
SOLID principles adapted for Go. While Go isn't a traditional OOP language, these principles improve code design through interfaces, composition, and package organization.
</overview>

<single_responsibility>
**Single Responsibility Principle (SRP)**

A package, type, or function should have one reason to change.

<guidance>
**At package level:**
- Package name describes its single purpose
- Avoid: `utils`, `common`, `helpers` (too many responsibilities)
- Good: `http`, `json`, `user`, `auth`

**At type level:**
- Each struct does one thing
- If you describe it with "and", split it

**At function level:**
- One action per function
- Functions that do A and B should call separate A() and B()
</guidance>

<example>
```go
// Bad: multiple responsibilities
type UserService struct {
    db *sql.DB
}

func (s *UserService) CreateAndEmail(u User) error {
    // creates user AND sends email
}

// Good: separated responsibilities
type UserStore struct {
    db *sql.DB
}

func (s *UserStore) Create(u User) error { ... }

type EmailSender struct {
    client SMTPClient
}

func (e *EmailSender) SendWelcome(u User) error { ... }

// Coordination happens at a higher level
func CreateUser(store *UserStore, email *EmailSender, u User) error {
    if err := store.Create(u); err != nil {
        return err
    }
    return email.SendWelcome(u)
}
```
</example>
</single_responsibility>

<open_closed>
**Open/Closed Principle (OCP)**

Software should be open for extension but closed for modification.

<guidance>
In Go, this means:
- Define behavior through interfaces
- Extend by implementing interfaces, not modifying existing code
- Use composition (embedding) to add functionality
</guidance>

<example>
```go
// Closed for modification: NotificationSender interface is fixed
type NotificationSender interface {
    Send(msg string) error
}

// Open for extension: add new implementations without changing existing code
type EmailNotifier struct { ... }
func (e *EmailNotifier) Send(msg string) error { ... }

type SMSNotifier struct { ... }
func (s *SMSNotifier) Send(msg string) error { ... }

type SlackNotifier struct { ... }  // new! no changes to existing code
func (s *SlackNotifier) Send(msg string) error { ... }

// Usage remains unchanged
func NotifyUser(sender NotificationSender, msg string) error {
    return sender.Send(msg)
}
```
</example>

<embedding>
```go
// Extension via embedding
type LoggingStore struct {
    UserStore           // embedded - gets all UserStore methods
    logger *log.Logger
}

// Override specific methods
func (s *LoggingStore) Create(u User) error {
    s.logger.Printf("creating user: %s", u.Name)
    return s.UserStore.Create(u)  // delegate to embedded
}
```
</embedding>
</open_closed>

<liskov_substitution>
**Liskov Substitution Principle (LSP)**

Implementations must be substitutable for their interfaces.

<guidance>
In Go:
- Interfaces are satisfied implicitly
- Any type with matching methods works
- Follow: "Require no more, promise no less"
</guidance>

<example>
```go
// io.Reader contract
type Reader interface {
    Read(p []byte) (n int, err error)
}

// All these are substitutable wherever Reader is accepted:
// - strings.Reader
// - bytes.Buffer
// - os.File
// - http.Response.Body
// - gzip.Reader

func ProcessData(r io.Reader) error {
    // Works with ANY Reader implementation
    data, err := io.ReadAll(r)
    // ...
}

// All calls are equivalent:
ProcessData(strings.NewReader("hello"))
ProcessData(&bytes.Buffer{})
ProcessData(file)
ProcessData(resp.Body)
```
</example>

<violations>
**Violations to avoid:**
```go
// Bad: implementation requires more than interface promises
type BadReader struct{}

func (r *BadReader) Read(p []byte) (int, error) {
    if len(p) < 100 {
        panic("buffer too small")  // violates LSP!
    }
    // ...
}
```
</violations>
</liskov_substitution>

<interface_segregation>
**Interface Segregation Principle (ISP)**

Clients should not depend on methods they don't use.

<guidance>
In Go:
- Keep interfaces small (1-2 methods ideal)
- Define interfaces where they're used, not where implemented
- The smaller the interface, the more implementations can satisfy it
</guidance>

<example>
```go
// Bad: large interface forces unnecessary dependencies
type Repository interface {
    Get(id string) (*User, error)
    List() ([]*User, error)
    Create(u *User) error
    Update(u *User) error
    Delete(id string) error
    Search(query string) ([]*User, error)
    Count() (int, error)
}

// A function that only reads doesn't need write methods:
func DisplayUser(repo Repository, id string) { ... }  // depends on 7 methods!

// Good: small, focused interfaces
type UserGetter interface {
    Get(id string) (*User, error)
}

type UserLister interface {
    List() ([]*User, error)
}

type UserWriter interface {
    Create(u *User) error
    Update(u *User) error
}

// Functions depend only on what they need
func DisplayUser(getter UserGetter, id string) { ... }  // depends on 1 method
func ListUsers(lister UserLister) { ... }
func SaveUser(writer UserWriter, u *User) { ... }
```
</example>

<composition>
```go
// Compose interfaces when needed
type UserStore interface {
    UserGetter
    UserLister
    UserWriter
}

// But consumers still depend on minimal interfaces
```
</composition>
</interface_segregation>

<dependency_inversion>
**Dependency Inversion Principle (DIP)**

High-level modules should not depend on low-level modules. Both should depend on abstractions.

<guidance>
In Go:
- "Accept interfaces, return structs"
- Define interfaces in the consumer package, not provider
- Push concrete types to `main` (the highest level)
</guidance>

<example>
```go
// Bad: high-level depends on low-level concrete type
package service

import "myapp/postgres"  // depends on specific implementation

type UserService struct {
    store *postgres.UserStore  // concrete dependency
}

// Good: depend on abstraction
package service

// Interface defined where it's used (not in postgres package)
type UserStore interface {
    Get(id string) (*User, error)
    Save(u *User) error
}

type UserService struct {
    store UserStore  // abstract dependency
}

// Concrete wiring happens in main
package main

func main() {
    // main (highest level) knows about concrete types
    pgStore := postgres.NewUserStore(db)
    svc := service.NewUserService(pgStore)
}
```
</example>

<import_direction>
**Import graph should flow downward:**

```
main (highest level - concrete implementations)
  ↓
handlers (depend on service interfaces)
  ↓
services (depend on store interfaces)
  ↓
domain (pure types, no dependencies)
```

Lower-level packages should never import higher-level packages.
</import_direction>
</dependency_inversion>

<summary>
**SOLID in Go Summary:**

| Principle | Go Implementation |
|-----------|-------------------|
| SRP | Small, focused packages and types |
| OCP | Interfaces + embedding for extension |
| LSP | Implicit interface satisfaction |
| ISP | Small interfaces (1-2 methods) |
| DIP | "Accept interfaces, return structs" |

**Key insight:** Interfaces are the foundation of SOLID in Go. Well-designed interfaces enable all five principles.
</summary>
