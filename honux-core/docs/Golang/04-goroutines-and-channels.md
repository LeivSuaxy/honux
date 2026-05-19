# 04 — Goroutines & Channels

> Goroutines are lightweight threads managed by the Go runtime. Channels are the idiomatic way to communicate between them. The Go mantra: **"Do not communicate by sharing memory; instead, share memory by communicating."**

---

## 4.1 Goroutines

A goroutine is launched with the `go` keyword. It runs concurrently in the same address space.

```go
package main

import (
    "fmt"
    "time"
)

func sayHello(name string) {
    fmt.Printf("Hello, %s!\n", name)
}

func main() {
    go sayHello("Alice")  // spawns a goroutine; does not block
    go sayHello("Bob")

    // main() exits without waiting — goroutines may not run
    time.Sleep(100 * time.Millisecond) // crude wait; use sync.WaitGroup in production
}
```

### sync.WaitGroup — the right way to wait

```go
import "sync"

func main() {
    var wg sync.WaitGroup

    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            fmt.Printf("worker %d done\n", id)
        }(i) // capture i explicitly
    }

    wg.Wait() // blocks until all goroutines call Done()
    fmt.Println("all workers finished")
}
```

---

## 4.2 Channels

Channels are typed conduits for sending and receiving values between goroutines.

```go
// Unbuffered channel — send blocks until receiver is ready (and vice versa)
ch := make(chan int)

go func() {
    ch <- 42  // send
}()

v := <-ch     // receive — blocks until value is available
fmt.Println(v) // 42
```

### Buffered Channels

```go
// Buffered — send only blocks when buffer is full
ch := make(chan string, 3)

ch <- "a"
ch <- "b"
ch <- "c"
// ch <- "d" would block here (buffer full)

fmt.Println(<-ch) // "a"
fmt.Println(<-ch) // "b"
```

### Closing Channels

```go
ch := make(chan int, 5)
for i := 0; i < 5; i++ {
    ch <- i
}
close(ch) // signals no more values will be sent

// Range over a closed channel drains it cleanly
for v := range ch {
    fmt.Println(v)
}

// Check if channel is closed
v, ok := <-ch
if !ok {
    fmt.Println("channel is closed and drained")
}
```

> Only the **sender** should close a channel. Closing a nil or already-closed channel causes a panic.

---

## 4.3 Directional Channels

Restricting channel direction in function signatures improves safety and documents intent.

```go
func producer(out chan<- int) { // send-only
    for i := 0; i < 5; i++ {
        out <- i
    }
    close(out)
}

func consumer(in <-chan int) { // receive-only
    for v := range in {
        fmt.Println("received:", v)
    }
}

func main() {
    ch := make(chan int, 5)
    go producer(ch)
    consumer(ch)
}
```

---

## 4.4 select — Multiplexing Channels

`select` waits on multiple channel operations, executing the first one that's ready.

```go
func main() {
    ch1 := make(chan string)
    ch2 := make(chan string)

    go func() { time.Sleep(1 * time.Second); ch1 <- "one" }()
    go func() { time.Sleep(2 * time.Second); ch2 <- "two" }()

    for i := 0; i < 2; i++ {
        select {
        case msg := <-ch1:
            fmt.Println("received from ch1:", msg)
        case msg := <-ch2:
            fmt.Println("received from ch2:", msg)
        }
    }
}
```

### Non-blocking operations with default

```go
select {
case v := <-ch:
    fmt.Println("got:", v)
default:
    fmt.Println("no value ready — moving on")
}
```

### Timeout pattern

```go
select {
case result := <-ch:
    fmt.Println("result:", result)
case <-time.After(2 * time.Second):
    fmt.Println("timeout — operation took too long")
}
```

---

## 4.5 Fan-Out & Fan-In

**Fan-out**: distribute work from one channel to many workers.  
**Fan-in**: merge multiple channels into one.

```go
// Fan-out: distribute jobs to N workers
func fanOut(jobs <-chan int, n int) []<-chan int {
    outputs := make([]<-chan int, n)
    for i := 0; i < n; i++ {
        out := make(chan int)
        outputs[i] = out
        go func(o chan<- int) {
            for j := range jobs {
                o <- j * j // example processing
            }
            close(o)
        }(out)
    }
    return outputs
}

// Fan-in: merge N channels into one
func fanIn(channels ...<-chan int) <-chan int {
    merged := make(chan int)
    var wg sync.WaitGroup

    forward := func(ch <-chan int) {
        defer wg.Done()
        for v := range ch {
            merged <- v
        }
    }

    wg.Add(len(channels))
    for _, ch := range channels {
        go forward(ch)
    }

    go func() {
        wg.Wait()
        close(merged)
    }()
    return merged
}
```

---

## 4.6 Pipeline Pattern

Pipelines chain stages where each stage reads from one channel and writes to another.

```go
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}

func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            out <- n * n
        }
        close(out)
    }()
    return out
}

func filter(in <-chan int, fn func(int) bool) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            if fn(n) {
                out <- n
            }
        }
        close(out)
    }()
    return out
}

func main() {
    // Pipeline: generate → square → filter(>10)
    nums := generate(1, 2, 3, 4, 5)
    squared := square(nums)
    big := filter(squared, func(n int) bool { return n > 10 })

    for v := range big {
        fmt.Println(v) // 16, 25
    }
}
```

---

## 4.7 Cancellation with context.Context

`context.Context` is the idiomatic way to propagate cancellation across goroutines.

```go
import "context"

func worker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():
            fmt.Printf("worker %d cancelled: %v\n", id, ctx.Err())
            return
        default:
            fmt.Printf("worker %d doing work...\n", id)
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel() // always call cancel to free resources

    var wg sync.WaitGroup
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            worker(ctx, id)
        }(i)
    }
    wg.Wait()
}
```

---

## 4.8 Mutex — Protecting Shared State

When goroutines must share memory (instead of communicating), use `sync.Mutex`.

```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *SafeCounter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}

func main() {
    c := &SafeCounter{}
    var wg sync.WaitGroup

    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.Increment()
        }()
    }
    wg.Wait()
    fmt.Println(c.Value()) // 1000
}
```

### sync.RWMutex — for read-heavy workloads

```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()         // multiple readers can hold RLock simultaneously
    defer c.mu.RUnlock()
    v, ok := c.items[key]
    return v, ok
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()          // exclusive write lock
    defer c.mu.Unlock()
    c.items[key] = value
}
```

---

## 4.9 sync.Once — Run Exactly Once

```go
type Singleton struct {
    data string
}

var (
    instance *Singleton
    once     sync.Once
)

func GetInstance() *Singleton {
    once.Do(func() {
        instance = &Singleton{data: "initialized"}
    })
    return instance
}
```

---

## 4.10 errgroup — Goroutines with Error Handling

```go
import "golang.org/x/sync/errgroup"

func main() {
    g, ctx := errgroup.WithContext(context.Background())

    urls := []string{"https://a.com", "https://b.com", "https://c.com"}

    for _, url := range urls {
        url := url // capture
        g.Go(func() error {
            return fetch(ctx, url)
        })
    }

    if err := g.Wait(); err != nil {
        fmt.Println("error:", err)
    }
}
```

---

## 4.11 Common Mistakes

```go
// ❌ Goroutine leak — nothing drains the channel; goroutine blocks forever
ch := make(chan int)
go func() {
    ch <- 42 // nobody receives — leaked goroutine
}()

// ✅ Use buffered channel or ensure a receiver exists

// ❌ Race condition — multiple goroutines write to a shared map
m := map[string]int{}
go func() { m["a"] = 1 }()
go func() { m["b"] = 2 }()

// ✅ Use sync.Mutex or sync.Map
var mu sync.Mutex
go func() { mu.Lock(); m["a"] = 1; mu.Unlock() }()
```

---

## 4.12 Concurrency Patterns Summary

| Pattern | Use When |
|---|---|
| Goroutine + WaitGroup | Fire and collect many independent tasks |
| Buffered channel | Decouple producer/consumer speed |
| Pipeline | Chain sequential processing stages |
| Fan-out / Fan-in | Parallelize work, merge results |
| select + done channel | Cancellable operations |
| context.Context | Propagate deadlines across call chains |
| sync.Mutex | Protect shared mutable state |
| sync.RWMutex | Read-heavy shared state |
| sync.Once | Singleton / lazy initialization |
| errgroup | Goroutines that can fail |

---

*Next: [05 — Interfaces →](./05-interfaces.md)*
