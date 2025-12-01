# Workflow: Write Concurrent Go Code

<required_reading>
**Read these reference files NOW:**
1. references/concurrency-patterns.md
2. references/effective-go.md
</required_reading>

<process>

<step name="determine-need">
**Confirm concurrency is needed:**

Concurrency adds complexity. Use it when:
- I/O operations can happen in parallel (network calls, file operations)
- CPU-bound work can be parallelized across cores
- You need to handle multiple independent tasks simultaneously
- Responsiveness requires background processing

Don't use concurrency:
- For simple sequential operations
- When the overhead outweighs benefits
- To make code "look concurrent"
</step>

<step name="choose-pattern">
**Select the appropriate concurrency pattern:**

| Pattern | Use When |
|---------|----------|
| Simple goroutine | Fire-and-forget background task |
| Goroutine with channel | Need result back from async work |
| Worker pool | Processing many similar tasks with bounded resources |
| Pipeline | Data flows through stages of transformation |
| Fan-out/Fan-in | Distribute work, then collect results |
| Select with timeout | Need to limit wait time |
</step>

<step name="use-context">
**Always use context.Context:**

```go
func Process(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()  // respect cancellation
        default:
            if err := processItem(ctx, item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

Context enables:
- Cancellation propagation
- Timeouts
- Request-scoped values
</step>

<step name="prevent-leaks">
**Ensure goroutines can exit:**

Every goroutine must have a way to terminate:

```go
// Bad: goroutine can leak if ch is never closed
go func() {
    for item := range ch {
        process(item)
    }
}()

// Good: context allows cancellation
go func() {
    for {
        select {
        case <-ctx.Done():
            return  // clean exit
        case item, ok := <-ch:
            if !ok {
                return  // channel closed
            }
            process(item)
        }
    }
}()
```

Use `make lint` and `make test` (includes race detector) to catch issues.
</step>

<step name="handle-channels-correctly">
**Follow channel best practices:**

```go
// Create appropriately sized channels
ch := make(chan Result)       // unbuffered: synchronous
ch := make(chan Result, 10)   // buffered: up to 10 pending

// Sender closes, not receiver
go func() {
    defer close(ch)  // signal completion
    for _, item := range items {
        ch <- process(item)
    }
}()

// Receiver ranges until closed
for result := range ch {
    handleResult(result)
}
```

Rules:
- Only the sender should close a channel
- Never close a channel from the receiver side
- Closing a nil channel panics
- Sending to a closed channel panics
</step>

<step name="protect-shared-state">
**Use appropriate synchronization:**

Prefer channels for passing data. Use mutexes for protecting shared state:

```go
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}
```

For read-heavy workloads, use `sync.RWMutex`.
</step>

<step name="implement-common-patterns">
**Worker Pool:**

```go
func WorkerPool(ctx context.Context, jobs <-chan Job, workers int) <-chan Result {
    results := make(chan Result)
    var wg sync.WaitGroup

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                select {
                case <-ctx.Done():
                    return
                case job, ok := <-jobs:
                    if !ok {
                        return
                    }
                    results <- process(job)
                }
            }
        }()
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    return results
}
```

**Fan-Out/Fan-In:**

```go
func FanOut(ctx context.Context, input <-chan int, workers int) []<-chan int {
    channels := make([]<-chan int, workers)
    for i := 0; i < workers; i++ {
        channels[i] = worker(ctx, input)
    }
    return channels
}

func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
    var wg sync.WaitGroup
    out := make(chan int)

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for n := range c {
                select {
                case out <- n:
                case <-ctx.Done():
                    return
                }
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}
```
</step>

<step name="handle-errors">
**Collect errors from goroutines:**

```go
// Using errgroup
import "golang.org/x/sync/errgroup"

func ProcessAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        item := item  // capture for goroutine
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()  // returns first error, cancels others
}
```
</step>

<step name="test-concurrent-code">
**Test with race detector:**

```bash
make test
```

Write tests that:
- Exercise concurrent paths
- Check for race conditions
- Verify goroutine cleanup
- Test cancellation behavior

```go
func TestWorkerPool_Cancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    jobs := make(chan Job, 100)

    results := WorkerPool(ctx, jobs, 4)

    // Cancel before completion
    cancel()

    // Verify workers exit and channel closes
    for range results {
        // drain
    }
}
```
</step>

</process>

<anti_patterns>
**Avoid:**
- Goroutines without exit conditions (leaks)
- Closing channels from receivers
- Sending to potentially closed channels without checking
- Using `time.Sleep` for synchronization
- Sharing memory without synchronization
- Ignoring context cancellation
- Creating unbounded goroutines (use worker pools)
</anti_patterns>

<success_criteria>
Concurrent code is correct when:
- [ ] All goroutines can exit (no leaks)
- [ ] context.Context is used for cancellation
- [ ] Channels are closed by senders only
- [ ] Shared state is protected
- [ ] `make test` passes
- [ ] Errors are propagated from goroutines
- [ ] The pattern matches the use case
</success_criteria>
