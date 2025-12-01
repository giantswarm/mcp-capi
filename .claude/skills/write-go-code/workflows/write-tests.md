# Workflow: Write Go Tests

<required_reading>
**Read these reference files NOW:**
1. references/testing-patterns.md
2. references/interfaces.md
</required_reading>

<process>

<step name="choose-testing-approach">
**Choose between standard testing and Ginkgo:**

Go offers two excellent testing approaches. Choose based on project context:

| Approach | Best For |
|----------|----------|
| **Standard `testing`** | Simple unit tests, projects preferring minimal dependencies, quick one-off tests |
| **Ginkgo/Gomega** | BDD-style specs, complex integration tests, projects already using Ginkgo |

If the project already uses one approach, stay consistent. Both are equally valid.
</step>

<step name="understand-what-to-test">
**Identify test requirements:**

Prioritize testing:
- Public API behavior (exported functions and methods)
- Error conditions and edge cases
- Business logic and rules
- Integration points (with fakes/mocks)

Don't test:
- Private implementation details
- Simple getters/setters
- Standard library functionality
</step>

<step name="use-table-driven-tests">
**Structure tests as tables:**

```go
func TestParse(t *testing.T) {
    tests := map[string]struct {
        input   string
        want    int
        wantErr bool
    }{
        "valid positive":    {input: "42", want: 42},
        "valid negative":    {input: "-7", want: -7},
        "zero":              {input: "0", want: 0},
        "empty string":      {input: "", wantErr: true},
        "non-numeric":       {input: "abc", wantErr: true},
        "overflow":          {input: "999999999999", wantErr: true},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            got, err := Parse(tc.input)

            if tc.wantErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tc.want {
                t.Errorf("got %d, want %d", got, tc.want)
            }
        })
    }
}
```

Benefits:
- Easy to add cases
- Clear test names in output
- Maps randomize order (exposes order-dependent bugs)
</step>

<step name="ginkgo-structure">
**Ginkgo alternative: Describe/Context/It:**

For Ginkgo projects, use BDD structure:

```go
var _ = Describe("Parser", func() {
    DescribeTable("parses input",
        func(input string, want int, wantErr bool) {
            got, err := Parse(input)
            if wantErr {
                Expect(err).To(HaveOccurred())
                return
            }
            Expect(err).NotTo(HaveOccurred())
            Expect(got).To(Equal(want))
        },
        Entry("valid positive", "42", 42, false),
        Entry("valid negative", "-7", -7, false),
        Entry("empty string", "", 0, true),
        Entry("non-numeric", "abc", 0, true),
    )
})
```

**Suite setup:**

```go
// parser_suite_test.go
package parser_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "testing"
)

func TestParser(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Parser Suite")
}
```
</step>

<step name="use-subtests">
**Always use t.Run for subtests:**

```go
func TestUser(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        // test creation
    })

    t.Run("Validate", func(t *testing.T) {
        t.Run("empty name", func(t *testing.T) {
            // nested subtest
        })
        t.Run("invalid email", func(t *testing.T) {
            // nested subtest
        })
    })
}
```

Run specific tests: `make test` (or `go test -run TestUser/Create` for a specific test)
</step>

<step name="create-test-doubles">
**Use interfaces for testability:**

```go
// Production code defines what it needs
type UserStore interface {
    Get(id string) (*User, error)
}

type Service struct {
    store UserStore
}

// Test provides a fake implementation
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

func TestService_GetUser(t *testing.T) {
    store := &fakeStore{
        users: map[string]*User{"1": {ID: "1", Name: "Alice"}},
    }
    svc := &Service{store: store}

    user, err := svc.GetUser("1")
    // ...
}
```

Prefer fakes over mocks - they're simpler and test real behavior.
</step>

<step name="test-errors">
**Test error conditions explicitly:**

```go
func TestService_GetUser_NotFound(t *testing.T) {
    store := &fakeStore{err: ErrNotFound}
    svc := &Service{store: store}

    _, err := svc.GetUser("nonexistent")

    if !errors.Is(err, ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

Test both happy path and error paths.
</step>

<step name="use-test-helpers">
**Create helpers for common setup:**

**Standard testing:**
```go
func testHelper(t *testing.T) *Service {
    t.Helper()  // Errors report caller's line
    store := &fakeStore{users: make(map[string]*User)}
    return &Service{store: store}
}

func TestService_CreateUser(t *testing.T) {
    svc := testHelper(t)
    // ...
}
```

**Ginkgo:**
```go
func mustCreateService() *Service {
    GinkgoHelper()  // Errors report caller's line
    store := &fakeStore{users: make(map[string]*User)}
    return &Service{store: store}
}

var _ = Describe("Service", func() {
    It("creates user", func() {
        svc := mustCreateService()
        // ...
    })
})
```

Use `t.Helper()` or `GinkgoHelper()` in helper functions for better error reporting.
</step>

<step name="test-concurrent-code">
**Test concurrency with race detector:**

```bash
make test
```

```go
func TestCounter_Concurrent(t *testing.T) {
    c := &Counter{}
    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.Inc()
        }()
    }

    wg.Wait()

    if got := c.Value(); got != 100 {
        t.Errorf("got %d, want 100", got)
    }
}
```
</step>

<step name="use-testing-conventions">
**Follow Go testing conventions:**

File naming:
- `foo.go` → `foo_test.go`
- Tests in same package or `_test` package

Function naming:
- `TestFunctionName` for tests
- `BenchmarkFunctionName` for benchmarks
- `ExampleFunctionName` for examples (appear in docs)

Test package:
- Same package: can test private functions
- `_test` package: tests only public API (preferred for libraries)
</step>

<step name="verify-coverage">
**Check test coverage:**

```bash
# Generate coverage report
make test-coverage
```

Aim for meaningful coverage, not 100%. Focus on critical paths.
</step>

</process>

<anti_patterns>
**Avoid:**
- Testing implementation details (brittle tests)
- Complex mock setups (use fakes instead)
- Tests that depend on execution order
- Sleeping for synchronization (`time.Sleep`)
- Ignoring test failures or skipping flaky tests
- Testing private functions extensively (test through public API)
- Not testing error paths
</anti_patterns>

<success_criteria>
Tests are good when:
- [ ] Table-driven for multiple cases (map-based or DescribeTable)
- [ ] Subtests used with descriptive names (t.Run or Context/It)
- [ ] Interfaces used for test doubles
- [ ] Both success and error paths tested
- [ ] Race detector passes (`make test`)
- [ ] Helpers use `t.Helper()` or `GinkgoHelper()`
- [ ] Coverage is meaningful for critical code
- [ ] Tests are independent (no shared state)
- [ ] Consistent with project's testing style (standard or Ginkgo)
</success_criteria>
