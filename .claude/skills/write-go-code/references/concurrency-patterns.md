<overview>
Go's concurrency model is built on goroutines and channels. The key philosophy: "Do not communicate by sharing memory; instead, share memory by communicating."
</overview>

<fundamentals>
**Goroutines:**

```go
go doWork()           // fire and forget
go func() { ... }()   // inline function
```

Goroutines are lightweight (~2KB stack). You can run thousands without issue.

**Channels:**

```go
ch := make(chan int)     // unbuffered (synchronous)
ch := make(chan int, 10) // buffered (async up to capacity)

ch <- value  // send
v := <-ch    // receive
close(ch)    // close (sender only!)
```

**Channel rules:**
- Send to nil channel blocks forever
- Receive from nil channel blocks forever
- Send to closed channel panics
- Receive from closed channel returns zero value immediately
- Only sender should close
</fundamentals>

<patterns>

<pattern name="worker-pool">
**Worker Pool**

Process many tasks with bounded concurrency:

```go
func WorkerPool(ctx context.Context, jobs <-chan Job, numWorkers int) <-chan Result {
    results := make(chan Result)
    var wg sync.WaitGroup

    for i := 0; i < numWorkers; i++ {
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

// Usage
jobs := make(chan Job, 100)
results := WorkerPool(ctx, jobs, 5)

for _, j := range jobList {
    jobs <- j
}
close(jobs)

for r := range results {
    // handle result
}
```
</pattern>

<pattern name="fan-out-fan-in">
**Fan-Out / Fan-In**

Distribute work, then collect results:

```go
// Fan-out: distribute to multiple workers
func FanOut(ctx context.Context, input <-chan int, workers int) []<-chan int {
    channels := make([]<-chan int, workers)
    for i := 0; i < workers; i++ {
        channels[i] = process(ctx, input)
    }
    return channels
}

// Fan-in: merge multiple channels
func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                select {
                case out <- v:
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
</pattern>

<pattern name="pipeline">
**Pipeline**

Chain processing stages:

```go
func Generate(ctx context.Context, nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case out <- n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func Square(ctx context.Context, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            select {
            case out <- n * n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

// Usage
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

nums := Generate(ctx, 1, 2, 3, 4, 5)
squared := Square(ctx, nums)

for n := range squared {
    fmt.Println(n)
}
```
</pattern>

<pattern name="semaphore">
**Semaphore (Bounded Concurrency)**

Limit concurrent operations:

```go
type Semaphore chan struct{}

func NewSemaphore(n int) Semaphore {
    return make(chan struct{}, n)
}

func (s Semaphore) Acquire() { s <- struct{}{} }
func (s Semaphore) Release() { <-s }

// Usage
sem := NewSemaphore(10)  // max 10 concurrent

for _, item := range items {
    sem.Acquire()
    go func(item Item) {
        defer sem.Release()
        process(item)
    }(item)
}
```
</pattern>

<pattern name="timeout">
**Timeout Pattern**

```go
func DoWithTimeout(ctx context.Context, timeout time.Duration) (Result, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    resultCh := make(chan Result, 1)
    errCh := make(chan error, 1)

    go func() {
        result, err := doWork(ctx)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- result
    }()

    select {
    case result := <-resultCh:
        return result, nil
    case err := <-errCh:
        return Result{}, err
    case <-ctx.Done():
        return Result{}, ctx.Err()
    }
}
```
</pattern>

<pattern name="broadcast">
**Broadcast (Multiple Receivers)**

```go
type Broadcaster struct {
    mu        sync.RWMutex
    listeners []chan Event
}

func (b *Broadcaster) Subscribe() <-chan Event {
    ch := make(chan Event, 10)
    b.mu.Lock()
    b.listeners = append(b.listeners, ch)
    b.mu.Unlock()
    return ch
}

func (b *Broadcaster) Broadcast(event Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.listeners {
        select {
        case ch <- event:
        default:
            // drop if listener is slow
        }
    }
}
```
</pattern>

</patterns>

<synchronization>
**sync.WaitGroup:**

```go
var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}

wg.Wait()  // blocks until all done
```

**sync.Mutex / sync.RWMutex:**

```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

// RWMutex for read-heavy workloads
type SafeCache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *SafeCache) Get(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[key]
}
```

**sync.Once:**

```go
var (
    instance *Service
    once     sync.Once
)

func GetService() *Service {
    once.Do(func() {
        instance = &Service{}
        instance.Init()
    })
    return instance
}
```

**sync/atomic:**

```go
var counter atomic.Int64

counter.Add(1)
counter.Store(0)
v := counter.Load()
```
</synchronization>

<errgroup>
**errgroup for Error Handling:**

```go
import "golang.org/x/sync/errgroup"

func ProcessAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        item := item  // capture for goroutine
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()  // returns first error, cancels ctx
}

// With bounded concurrency
func ProcessAllBounded(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10)  // max 10 concurrent

    for _, item := range items {
        item := item
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()
}
```
</errgroup>

<context_usage>
**Context for Cancellation:**

```go
func Worker(ctx context.Context, jobs <-chan Job) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()  // respect cancellation
        case job, ok := <-jobs:
            if !ok {
                return nil  // channel closed
            }
            if err := process(ctx, job); err != nil {
                return err
            }
        }
    }
}

// Always pass context through
func process(ctx context.Context, job Job) error {
    // Check periodically in long operations
    for i := 0; i < len(job.Items); i++ {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        // do work
    }
    return nil
}
```
</context_usage>

<best_practices>
**Best Practices:**

1. **Always provide exit mechanism** - context or done channel
2. **Sender closes channels** - never receiver
3. **Use context for cancellation** - propagate through call chain
4. **Avoid shared memory** - use channels to pass data
5. **Test with race detector** - `make test`
6. **Bound goroutine count** - use worker pools
7. **Handle errors from goroutines** - use errgroup or error channels

**When to use what:**

| Need | Use |
|------|-----|
| Simple async | goroutine + channel |
| Multiple workers | worker pool |
| Error collection | errgroup |
| State protection | mutex |
| Simple counter | atomic |
| One-time init | sync.Once |
| Wait for completion | WaitGroup or channel |
</best_practices>

<anti_patterns>
**Avoid:**

```go
// Goroutine leak - no exit path
go func() {
    for range ch { }  // blocks forever if ch never closes
}()

// Sleep for sync
go work()
time.Sleep(time.Second)

// Closing from receiver
go func() {
    for v := range ch {
        close(ch)  // panic!
    }
}()

// Ignoring context
func work(ctx context.Context) {
    // never checks ctx.Done()
}
```
</anti_patterns>
