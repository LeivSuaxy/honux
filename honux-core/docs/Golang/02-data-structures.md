# 02 — Data Structures in Go

> Go's standard library and built-in types cover the most essential data structures. This guide covers the fundamentals every Go developer needs to know and use daily.

---

## 2.1 Arrays

Arrays in Go have a **fixed size** set at compile time. They are value types — assigned or passed by copy.

```go
var a [3]int              // [0 0 0]
b := [3]int{1, 2, 3}     // literal
c := [...]int{4, 5, 6}   // compiler counts elements

fmt.Println(len(b)) // 3
fmt.Println(b[1])   // 2

// Iteration
for i, v := range b {
    fmt.Printf("index=%d value=%d\n", i, v)
}
```

Arrays are rarely used directly in Go — **slices** are almost always preferred.

---

## 2.2 Slices

A slice is a **dynamic view** over an underlying array. It has three fields internally: pointer, length, capacity.

```go
// Creating slices
s1 := []int{1, 2, 3}
s2 := make([]int, 5)        // len=5, cap=5, all zeros
s3 := make([]int, 3, 10)    // len=3, cap=10

// Appending
s1 = append(s1, 4, 5)
fmt.Println(s1) // [1 2 3 4 5]

// Slicing (sub-slices share the backing array)
sub := s1[1:3]
fmt.Println(sub) // [2 3]

// Length and capacity
fmt.Println(len(s1), cap(s1))

// Copying — creates an independent slice
dst := make([]int, len(s1))
copy(dst, s1)
```

### Common Slice Patterns

```go
// Remove element at index i (order-preserving)
func remove(s []int, i int) []int {
    return append(s[:i], s[i+1:]...)
}

// Remove element at index i (fast, changes order)
func removeFast(s []int, i int) []int {
    s[i] = s[len(s)-1]
    return s[:len(s)-1]
}

// Filter
func filter(s []int, fn func(int) bool) []int {
    out := s[:0] // reuse the same backing array
    for _, v := range s {
        if fn(v) {
            out = append(out, v)
        }
    }
    return out
}
```

---

## 2.3 Maps

Maps are Go's built-in **hash table**. Keys must be comparable types.

```go
// Creation
m := map[string]int{}                    // empty
m2 := map[string]int{"a": 1, "b": 2}    // literal
m3 := make(map[string]int)               // empty, ready to use

// Insert / update
m["key"] = 42

// Read
v := m["key"]      // returns zero value if key doesn't exist
v, ok := m["key"]  // ok=false if missing — always prefer this form
if !ok {
    fmt.Println("key not found")
}

// Delete
delete(m, "key")

// Iterate (order is NOT guaranteed)
for k, v := range m2 {
    fmt.Printf("%s: %d\n", k, v)
}

// Length
fmt.Println(len(m2))
```

### Grouping with Maps

```go
words := []string{"apple", "banana", "avocado", "blueberry"}

// Group by first letter
byLetter := make(map[byte][]string)
for _, w := range words {
    byLetter[w[0]] = append(byLetter[w[0]], w)
}
// map[97:[apple avocado] 98:[banana blueberry]]
```

### Counting with Maps

```go
text := []string{"go", "is", "great", "go", "is", "fast"}
freq := make(map[string]int)
for _, w := range text {
    freq[w]++
}
// map[go:2 is:2 great:1 fast:1]
```

---

## 2.4 Strings

Strings in Go are **immutable byte slices** encoded in UTF-8.

```go
s := "Hello, 世界"

fmt.Println(len(s))         // byte length: 13
fmt.Println([]byte(s))      // raw bytes

// Iterate over Unicode code points (runes)
for i, r := range s {
    fmt.Printf("index=%d rune=%c\n", i, r)
}

// Convert between string and []byte
b := []byte(s)
s2 := string(b)

// String builder — efficient concatenation
var sb strings.Builder
for i := 0; i < 5; i++ {
    sb.WriteString("go")
}
fmt.Println(sb.String()) // "gogogogogo"
```

---

## 2.5 Stack (using a slice)

Go doesn't have a built-in stack. Use a slice.

```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(v T) {
    s.items = append(s.items, v)
}

func (s *Stack[T]) Pop() (T, bool) {
    var zero T
    if len(s.items) == 0 {
        return zero, false
    }
    top := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return top, true
}

func (s *Stack[T]) Peek() (T, bool) {
    var zero T
    if len(s.items) == 0 {
        return zero, false
    }
    return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int { return len(s.items) }

// Usage
func main() {
    s := &Stack[int]{}
    s.Push(1)
    s.Push(2)
    s.Push(3)
    v, _ := s.Pop()
    fmt.Println(v) // 3
}
```

---

## 2.6 Queue (using a slice)

```go
type Queue[T any] struct {
    items []T
}

func (q *Queue[T]) Enqueue(v T) {
    q.items = append(q.items, v)
}

func (q *Queue[T]) Dequeue() (T, bool) {
    var zero T
    if len(q.items) == 0 {
        return zero, false
    }
    front := q.items[0]
    q.items = q.items[1:]
    return front, true
}

func (q *Queue[T]) Len() int { return len(q.items) }
```

> For high-throughput queues, use a **ring buffer** or `container/ring` to avoid repeated allocations.

---

## 2.7 Linked List — `container/list`

Go's standard library includes a doubly linked list.

```go
import "container/list"

l := list.New()
e1 := l.PushBack(1)
e2 := l.PushBack(2)
l.PushFront(0)
l.InsertAfter(1.5, e1)

// Iterate
for e := l.Front(); e != nil; e = e.Next() {
    fmt.Println(e.Value)
}

l.Remove(e2)
fmt.Println(l.Len()) // 3
```

---

## 2.8 Heap (Priority Queue) — `container/heap`

```go
import "container/heap"

// MinHeap of ints
type IntHeap []int

func (h IntHeap) Len() int            { return len(h) }
func (h IntHeap) Less(i, j int) bool  { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *IntHeap) Push(x any)         { *h = append(*h, x.(int)) }
func (h *IntHeap) Pop() any {
    old := *h
    n := len(old)
    x := old[n-1]
    *h = old[:n-1]
    return x
}

func main() {
    h := &IntHeap{5, 3, 8, 1}
    heap.Init(h)

    heap.Push(h, 2)
    fmt.Println(heap.Pop(h)) // 1 — smallest element
    fmt.Println(heap.Pop(h)) // 2
}
```

---

## 2.9 Set (using a map)

Go doesn't have a native set. Use `map[T]struct{}` — `struct{}` has zero size.

```go
type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) Set[T] {
    s := make(Set[T])
    for _, v := range items {
        s.Add(v)
    }
    return s
}

func (s Set[T]) Add(v T)            { s[v] = struct{}{} }
func (s Set[T]) Remove(v T)         { delete(s, v) }
func (s Set[T]) Has(v T) bool       { _, ok := s[v]; return ok }
func (s Set[T]) Len() int           { return len(s) }

func (s Set[T]) Intersection(other Set[T]) Set[T] {
    result := make(Set[T])
    for k := range s {
        if other.Has(k) {
            result.Add(k)
        }
    }
    return result
}

// Usage
func main() {
    a := NewSet("go", "rust", "python")
    b := NewSet("rust", "java", "go")
    fmt.Println(a.Intersection(b)) // map[go:{} rust:{}]
}
```

---

## 2.10 Summary Table

| Structure | Built-in / Package | Use When |
|---|---|---|
| Array | Built-in | Fixed-size, stack-allocated data |
| Slice | Built-in | Most sequences; prefer over arrays |
| Map | Built-in | Key-value lookup, counting, grouping |
| String | Built-in | Immutable text |
| Stack | Custom slice | LIFO order |
| Queue | Custom slice | FIFO order |
| Linked List | `container/list` | Frequent insertions/removals in the middle |
| Heap / Priority Queue | `container/heap` | Min/max access in O(log n) |
| Set | `map[T]struct{}` | Membership testing, deduplication |

---

*Next: [03 — Structs & Methods →](./03-structs-and-methods.md)*
