<overview>
Go offers two excellent testing approaches: the built-in `testing` package for simple, direct tests, and Ginkgo/Gomega for BDD-style tests with rich structure. Both are equally valid—choose based on project needs and team preference.

**Standard testing**: Simple, no dependencies, great for straightforward unit tests.
**Ginkgo/Gomega**: Rich structure, readable specs, excellent for complex scenarios and integration tests.
</overview>

<basics>
**Test file naming:**
- `foo.go` → `foo_test.go`
- Same directory as code being tested

**Test function naming:**
- `TestFunctionName(t *testing.T)`
- `TestType_Method(t *testing.T)`
- `Test_unexported(t *testing.T)` (underscore for unexported)

**Running tests:**
```bash
make test                  # all tests (includes race detector)
make test-coverage         # with coverage report
```
</basics>

<table_driven>
**Table-Driven Tests:**

```go
func TestAdd(t *testing.T) {
    tests := map[string]struct {
        a, b int
        want int
    }{
        "positive":      {a: 2, b: 3, want: 5},
        "negative":      {a: -1, b: -1, want: -2},
        "mixed":         {a: -1, b: 1, want: 0},
        "zero":          {a: 0, b: 0, want: 0},
        "large numbers": {a: 1000000, b: 1000000, want: 2000000},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            got := Add(tc.a, tc.b)
            if got != tc.want {
                t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
            }
        })
    }
}
```

**Why use maps over slices:**
- Test names are explicit keys
- Iteration order is random (catches order-dependent bugs)
- Easier to navigate in large test tables

**With error cases:**

```go
func TestParse(t *testing.T) {
    tests := map[string]struct {
        input   string
        want    int
        wantErr bool
    }{
        "valid":     {input: "42", want: 42},
        "negative":  {input: "-7", want: -7},
        "empty":     {input: "", wantErr: true},
        "invalid":   {input: "abc", wantErr: true},
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
</table_driven>

<ginkgo_framework>
**Ginkgo BDD Framework:**

Ginkgo provides structured, readable tests using Describe/Context/It hierarchy:

```go
package mypackage_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "testing"
)

func TestSuite(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "MyPackage Suite")
}

var _ = Describe("Component", func() {
    var subject *Component

    BeforeEach(func() {
        subject = NewComponent()
    })

    Context("when initialized", func() {
        It("has default state", func() {
            Expect(subject.State()).To(Equal("ready"))
        })
    })

    Context("when processing", func() {
        BeforeEach(func() {
            subject.Process()
        })

        It("updates state", func() {
            Expect(subject.State()).To(Equal("processed"))
        })

        It("increments counter", func() {
            Expect(subject.Counter()).To(Equal(1))
        })
    })
})
```

**Setup and Teardown Nodes:**

```go
var _ = Describe("Database Tests", func() {
    var db *Database

    // Runs once before all tests in this Describe
    BeforeSuite(func() {
        // Global setup (e.g., start test database)
    })

    // Runs once after all tests in this Describe
    AfterSuite(func() {
        // Global teardown
    })

    // Runs before each It in this scope and nested scopes
    BeforeEach(func() {
        db = NewDatabase()
    })

    // Runs after each It
    AfterEach(func() {
        db.Close()
    })

    It("connects successfully", func() {
        Expect(db.Connect()).To(Succeed())
    })
})
```

**DeferCleanup for Scoped Cleanup:**

```go
var _ = Describe("Resource Tests", func() {
    It("creates temporary file", func() {
        f, err := os.CreateTemp("", "test")
        Expect(err).NotTo(HaveOccurred())

        // Cleanup runs after this It completes
        DeferCleanup(func() {
            os.Remove(f.Name())
        })

        // Use file...
    })

    It("with cleanup arguments", func() {
        conn := OpenConnection()
        DeferCleanup(conn.Close)  // Method value
    })
})
```

**Closure Variables for State Sharing:**

```go
var _ = Describe("User Service", func() {
    var (
        svc   *UserService
        store *FakeStore
        ctx   context.Context
    )

    BeforeEach(func() {
        ctx = context.Background()
        store = NewFakeStore()
        svc = NewUserService(store)
    })

    Context("when user exists", func() {
        BeforeEach(func() {
            store.AddUser(&User{ID: "1", Name: "Alice"})
        })

        It("returns the user", func() {
            user, err := svc.Get(ctx, "1")
            Expect(err).NotTo(HaveOccurred())
            Expect(user.Name).To(Equal("Alice"))
        })
    })
})
```

**Focus and Pending:**

```go
// Focus: run only these tests (remove before commit!)
FDescribe("focused describe", func() { ... })
FContext("focused context", func() { ... })
FIt("focused test", func() { ... })

// Pending: skip these tests
PDescribe("pending describe", func() { ... })
PContext("pending context", func() { ... })
PIt("pending test", func() { ... })

// Skip programmatically
It("conditional skip", func() {
    if runtime.GOOS == "windows" {
        Skip("not supported on Windows")
    }
    // ...
})
```

**GinkgoHelper for Test Helpers:**

```go
func mustCreateUser(name string) *User {
    GinkgoHelper()  // Errors report caller's line
    user, err := CreateUser(name)
    Expect(err).NotTo(HaveOccurred())
    return user
}

var _ = Describe("Users", func() {
    It("processes user", func() {
        user := mustCreateUser("Alice")  // Error points here, not inside helper
        // ...
    })
})
```
</ginkgo_framework>

<ginkgo_tables>
**Table-Driven Tests with DescribeTable:**

```go
var _ = Describe("Validator", func() {
    DescribeTable("validates input",
        func(input string, expected bool) {
            Expect(Validate(input)).To(Equal(expected))
        },
        Entry("valid input", "good", true),
        Entry("empty input", "", false),
        Entry("invalid chars", "bad!", false),
        Entry("too long", strings.Repeat("a", 101), false),
    )

    // With description formatting
    DescribeTable("parses numbers",
        func(input string, expected int, shouldErr bool) {
            result, err := Parse(input)
            if shouldErr {
                Expect(err).To(HaveOccurred())
            } else {
                Expect(err).NotTo(HaveOccurred())
                Expect(result).To(Equal(expected))
            }
        },
        Entry("positive: 42", "42", 42, false),
        Entry("negative: -7", "-7", -7, false),
        Entry("invalid: abc", "abc", 0, true),
    )
})
```

**DescribeTableSubtree for Complex Scenarios:**

```go
var _ = Describe("API Client", func() {
    DescribeTableSubtree("with different auth methods",
        func(authType string, setupAuth func(*Client)) {
            var client *Client

            BeforeEach(func() {
                client = NewClient()
                setupAuth(client)
            })

            It("authenticates successfully", func() {
                Expect(client.Authenticate()).To(Succeed())
            })

            It("can make requests", func() {
                resp, err := client.Get("/api/status")
                Expect(err).NotTo(HaveOccurred())
                Expect(resp.StatusCode).To(Equal(200))
            })
        },
        Entry("basic auth", "basic", func(c *Client) {
            c.SetBasicAuth("user", "pass")
        }),
        Entry("token auth", "token", func(c *Client) {
            c.SetToken("test-token")
        }),
    )
})
```
</ginkgo_tables>

<gomega_matchers>
**Gomega Matchers:**

**Basic Assertions:**

```go
Expect(value).To(Equal(expected))           // Deep equality
Expect(value).NotTo(Equal(other))           // Negation
Expect(value).To(BeNil())                   // Nil check
Expect(value).NotTo(BeNil())                // Not nil
Expect(value).To(BeZero())                  // Zero value
Expect(value).To(BeTrue())                  // Boolean true
Expect(value).To(BeFalse())                 // Boolean false
```

**Error Handling:**

```go
Expect(err).To(HaveOccurred())              // Error is not nil
Expect(err).NotTo(HaveOccurred())           // Error is nil
Expect(err).To(Succeed())                   // Same as NotTo(HaveOccurred())
Expect(err).To(MatchError("expected msg"))  // Error message
Expect(err).To(MatchError(ContainSubstring("partial")))
```

**Collections:**

```go
Expect(slice).To(BeEmpty())                 // Empty slice/map
Expect(slice).To(HaveLen(3))                // Length check
Expect(slice).To(ContainElement("item"))    // Contains element
Expect(slice).To(ContainElements("a", "b")) // Contains all
Expect(slice).To(ConsistOf("a", "b", "c"))  // Exactly these elements
Expect(map).To(HaveKey("key"))              // Map has key
Expect(map).To(HaveKeyWithValue("k", "v"))  // Map has key-value pair
```

**Strings:**

```go
Expect(str).To(ContainSubstring("partial"))
Expect(str).To(HavePrefix("start"))
Expect(str).To(HaveSuffix("end"))
Expect(str).To(MatchRegexp(`\d+`))
```

**Numeric Comparisons:**

```go
Expect(n).To(BeNumerically(">", 5))
Expect(n).To(BeNumerically(">=", 5))
Expect(n).To(BeNumerically("~", 5.0, 0.01))  // Within tolerance
```

**Eventually/Consistently for Async:**

```go
// Poll until condition is met (default: 1s timeout, 10ms interval)
Eventually(func() string {
    return client.Status()
}).Should(Equal("ready"))

// With custom timing
Eventually(func() error {
    return client.Ping()
}, "5s", "100ms").Should(Succeed())

// With context
Eventually(func(ctx context.Context) error {
    return client.PingWithContext(ctx)
}).WithContext(ctx).Should(Succeed())

// Verify condition stays true
Consistently(func() int {
    return counter.Value()
}, "1s", "100ms").Should(Equal(5))
```

**Combining Matchers:**

```go
Expect(value).To(And(
    BeNumerically(">", 0),
    BeNumerically("<", 100),
))

Expect(value).To(Or(
    Equal("active"),
    Equal("pending"),
))

Expect(value).To(Not(BeEmpty()))
```
</gomega_matchers>

<project_patterns>
**Project-Specific Testing Patterns:**

**Fluent API for Test Setup:**

This project uses a fluent API pattern for readable, chainable test setup with standard Go tests:

```go
func TestCapiListClusters(t *testing.T) {
    t.Parallel()

    t.Run("lists clusters in namespace", func(t *testing.T) {
        t.Parallel()
        harness.New(t).
            CreateNamespace("test-ns").
            CreateClusters("test-ns", "cluster-1", "cluster-2").
            ToolCall("capi_list_clusters").
            WithArg("namespace", "test-ns").
            AssertContent("expected.golden").
            Execute()
    })

    t.Run("handles empty namespace", func(t *testing.T) {
        t.Parallel()
        harness.New(t).
            CreateNamespace("empty-ns").
            ToolCall("capi_list_clusters").
            WithArg("namespace", "empty-ns").
            AssertContent("empty.golden").
            Execute()
    })
}
```

**Golden File Testing:**

Compare output against expected files for format consistency:

```go
// Compare actual output against golden file
h.ToolCall("tool_name").
    WithArg("key", "value").
    AssertContent("scenario.golden")

// Update golden files when output intentionally changes:
// UPDATE_GOLDEN=true make test
```

Golden files live in `testdata/` directories and should be committed to version control.

**k8senv for Kubernetes API Testing:**

Use k8senv for realistic Kubernetes API testing with kine-backed API servers:

```go
// TestMain initializes the k8senv manager (pool of API servers)
func TestMain(m *testing.M) {
    harness.InitManager()
    code := m.Run()
    harness.ShutdownManager()
    os.Exit(code)
}

func TestKubernetesIntegration(t *testing.T) {
    t.Parallel()

    t.Run("creates cluster resource", func(t *testing.T) {
        t.Parallel()
        harness.New(t).
            CreateNamespace("test-ns").
            CreateCluster("test-ns", "test-cluster").
            ToolCall("capi_list_clusters").
            WithArg("namespace", "test-ns").
            AssertContent("cluster_created.golden").
            Execute()
    })
}
```

**Test Isolation Patterns:**

Each test gets an isolated k8senv instance from the pool, preventing interference:

```go
t.Run("uses isolated API server per test", func(t *testing.T) {
    t.Parallel()
    // harness.New acquires a k8senv instance from the pool
    // Each instance has its own kube-apiserver + kine
    // Resources created here are fully isolated
    harness.New(t).
        CreateNamespace("test-ns").
        CreateClusters("test-ns", "cluster-1").
        // ...
        Execute()
    // Instance is released back to pool on cleanup
})
```
</project_patterns>

<subtests>
**Subtests with t.Run:**

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

// Run specific subtest:
// go test -run TestUser/Validate/empty_name (or use make test for all)
```

**Parallel subtests:**

```go
func TestParallel(t *testing.T) {
    tests := map[string]struct{ input string }{
        "case1": {input: "a"},
        "case2": {input: "b"},
    }

    for name, tc := range tests {
        tc := tc  // capture! critical for parallel
        t.Run(name, func(t *testing.T) {
            t.Parallel()  // run in parallel
            // use tc.input
        })
    }
}
```
</subtests>

<test_helpers>
**Test Helpers:**

Use `t.Helper()` in standard tests or `GinkgoHelper()` in Ginkgo tests to improve error reporting:

```go
// Standard testing - call t.Helper() first
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()  // errors report caller's line, not this line

    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("failed to open db: %v", err)
    }

    t.Cleanup(func() {
        db.Close()
    })

    return db
}

func TestWithDB(t *testing.T) {
    db := setupTestDB(t)
    // use db...
}
```

**t.Cleanup:**

```go
func TestWithTempFile(t *testing.T) {
    f, err := os.CreateTemp("", "test")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        os.Remove(f.Name())
    })

    // use f...
}
```
</test_helpers>

<test_doubles>
**Test Doubles (Fakes):**

```go
// Production interface
type UserStore interface {
    Get(id string) (*User, error)
    Save(u *User) error
}

// Fake implementation for tests
type fakeUserStore struct {
    users map[string]*User
    err   error
}

func (f *fakeUserStore) Get(id string) (*User, error) {
    if f.err != nil {
        return nil, f.err
    }
    u, ok := f.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return u, nil
}

func (f *fakeUserStore) Save(u *User) error {
    if f.err != nil {
        return f.err
    }
    f.users[u.ID] = u
    return nil
}

// Usage in test
func TestService_GetUser(t *testing.T) {
    store := &fakeUserStore{
        users: map[string]*User{
            "1": {ID: "1", Name: "Alice"},
        },
    }
    svc := NewService(store)

    user, err := svc.GetUser("1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "Alice" {
        t.Errorf("got name %q, want Alice", user.Name)
    }
}

func TestService_GetUser_NotFound(t *testing.T) {
    store := &fakeUserStore{users: map[string]*User{}}
    svc := NewService(store)

    _, err := svc.GetUser("nonexistent")
    if !errors.Is(err, ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

Prefer fakes over mocks - they're simpler and test real behavior.
</test_doubles>

<assertions>
**Go's Standard Assertion Pattern:**

```go
// Use if/t.Error or if/t.Fatal
if got != want {
    t.Errorf("Got() = %v, want %v", got, want)
}

if err != nil {
    t.Fatalf("unexpected error: %v", err)
}

// t.Error: record failure, continue
// t.Fatal: record failure, stop this test
```

**For complex comparisons:**

```go
import "github.com/google/go-cmp/cmp"

func TestComplex(t *testing.T) {
    got := GetConfig()
    want := Config{Name: "test", Values: []int{1, 2, 3}}

    if diff := cmp.Diff(want, got); diff != "" {
        t.Errorf("GetConfig() mismatch (-want +got):\n%s", diff)
    }
}
```
</assertions>

<testing_errors>
**Testing Error Conditions:**

```go
func TestParse_Error(t *testing.T) {
    tests := map[string]struct {
        input   string
        wantErr error
    }{
        "empty":   {input: "", wantErr: ErrEmpty},
        "invalid": {input: "xxx", wantErr: ErrInvalid},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            _, err := Parse(tc.input)

            if !errors.Is(err, tc.wantErr) {
                t.Errorf("got error %v, want %v", err, tc.wantErr)
            }
        })
    }
}
```
</testing_errors>

<benchmarks>
**Benchmarks:**

```go
func BenchmarkFoo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Foo()
    }
}

// With setup
func BenchmarkFooWithSetup(b *testing.B) {
    data := generateTestData()
    b.ResetTimer()  // don't count setup time

    for i := 0; i < b.N; i++ {
        Foo(data)
    }
}

// Run: go test -bench=. -benchmem (benchmarks run separately from make test)
```
</benchmarks>

<examples>
**Example Tests (appear in docs):**

```go
func ExampleHello() {
    fmt.Println(Hello("World"))
    // Output: Hello, World!
}

func ExampleUser_String() {
    u := User{Name: "Alice"}
    fmt.Println(u)
    // Output: User(Alice)
}
```

Examples are compiled and run as tests, and appear in documentation.
</examples>

<coverage>
**Coverage:**

```bash
# Generate coverage report
make test-coverage
```

Focus on meaningful coverage (business logic, error paths), not 100%.
</coverage>

<best_practices>
**Best Practices:**

1. **Use table-driven tests** - easy to add cases
2. **Use subtests (t.Run)** - better organization, selective running
3. **Use t.Helper()** - cleaner error locations
4. **Use t.Cleanup()** - guaranteed cleanup
5. **Prefer fakes over mocks** - simpler, test real behavior
6. **Test error paths** - not just happy path
7. **Run make test** - includes race detection
8. **Keep tests independent** - no shared mutable state

**Test file structure:**

```go
func TestFoo(t *testing.T) { ... }      // unit tests
func TestFoo_Integration(t *testing.T) { // integration tests
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // ...
}
func BenchmarkFoo(b *testing.B) { ... } // benchmarks
func ExampleFoo() { ... }               // examples
```
</best_practices>

<anti_patterns>
**Avoid:**

```go
// Testing private details (brittle)
func Test_internalHelper(t *testing.T) { }

// Shared state between tests
var testDB *sql.DB  // tests depend on each other

// Sleep for timing
time.Sleep(100 * time.Millisecond)

// Complex mocking setups
// (prefer simple fakes)

// No assertion message
if got != want {
    t.Fail()  // Bad: no info
}

// Testing through public API from _test package
// but testing implementation details
```
</anti_patterns>
