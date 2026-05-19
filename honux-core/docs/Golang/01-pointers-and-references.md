# 01 — Pointers & References in Go

> Go is not a reference-heavy language by design. Everything is **passed by value** unless you explicitly use a pointer. Understanding this distinction is fundamental to writing correct and efficient Go code.

---

## 1.1 Values vs Pointers

When you pass a variable to a function, Go copies the value. The function works on the copy, not the original.

```go
package main

import "fmt"

func double(n int) {
    n = n * 2 // modifies the local copy only
}

func doublePtr(n *int) {
    *n = *n * 2 // modifies the original value via pointer
}

func main() {
    x := 10
    double(x)
    fmt.Println(x) // 10 — unchanged

    doublePtr(&x)
    fmt.Println(x) // 20 — changed
}
```

---

## 1.2 Declaring Pointers

A pointer holds the **memory address** of a value. The type `*T` is a pointer to a value of type `T`.

```go
var p *int         // nil pointer (zero value for pointers)
x := 42
p = &x             // & gives the address of x

fmt.Println(p)     // 0xc0000b4010 (some memory address)
fmt.Println(*p)    // 42 — dereferencing: reading the value at the address
```

### `new` built-in

`new(T)` allocates memory for a `T`, zeroes it, and returns a `*T`.

```go
p := new(int)      // *int pointing to a zero int
*p = 7
fmt.Println(*p)    // 7
```

---

## 1.3 Pointer Receivers vs Value Receivers

This is the most practical use of pointers in Go — on struct methods.

```go
type Counter struct {
    count int
}

// Value receiver — operates on a copy. Mutation is NOT visible to the caller.
func (c Counter) ValueIncrement() {
    c.count++
}

// Pointer receiver — operates on the original. Mutation IS visible to the caller.
func (c *Counter) PtrIncrement() {
    c.count++
}

func main() {
    c := Counter{}

    c.ValueIncrement()
    fmt.Println(c.count) // 0 — no change

    c.PtrIncrement()
    fmt.Println(c.count) // 1 — changed
}
```

**Rule of thumb:** Use pointer receivers when:
- The method needs to mutate the receiver.
- The struct is large and copying would be expensive.
- Consistency: if any method uses a pointer receiver, use it on all methods of that type.

---

## 1.4 Nil Pointers

A pointer that hasn't been assigned a value is `nil`. Dereferencing a nil pointer causes a **panic**.

```go
var p *int
fmt.Println(p)  // <nil>
fmt.Println(*p) // PANIC: runtime error: invalid memory address or nil pointer dereference
```

Always guard against nil:

```go
func safeRead(p *int) int {
    if p == nil {
        return 0
    }
    return *p
}
```

---

## 1.5 Pointers to Structs

Go automatically dereferences struct pointers, so you don't need `(*p).Field` syntax.

```go
type Point struct {
    X, Y int
}

p := &Point{X: 1, Y: 2}
p.X = 10       // Go auto-dereferences; same as (*p).X = 10
fmt.Println(*p) // {10 2}
```

---

## 1.6 Slices, Maps, and Channels are Already Reference Types

These built-in types contain an internal pointer to their data. Passing them to a function does **not** copy the underlying data — the function shares the same backing array/map/channel.

```go
func appendItem(s []int) {
    s[0] = 999 // mutates the original backing array
}

func main() {
    nums := []int{1, 2, 3}
    appendItem(nums)
    fmt.Println(nums[0]) // 999
}
```

> **However**, `append` may allocate a new backing array. If you need the caller to see the new slice header (length + capacity), pass a `*[]int` or return the new slice.

```go
func grow(s *[]int) {
    *s = append(*s, 42)
}

func main() {
    nums := []int{1, 2, 3}
    grow(&nums)
    fmt.Println(nums) // [1 2 3 42]
}
```

---

## 1.7 Escape Analysis — Stack vs Heap

Go's compiler decides whether a variable lives on the **stack** (fast, automatically freed) or the **heap** (managed by GC).

- If a variable's address is returned or passed to another goroutine, it **escapes to the heap**.
- You rarely need to think about this manually, but it's useful for performance tuning.

```go
// x escapes to heap because its address is returned
func newInt() *int {
    x := 42
    return &x  // safe in Go; compiler allocates x on the heap
}
```

To inspect escape analysis:

```bash
go build -gcflags="-m" ./...
```

---

## 1.8 Quick Reference Table

| Concept | Syntax | Notes |
|---|---|---|
| Get address | `&x` | Returns `*T` |
| Dereference | `*p` | Returns the value at address |
| Nil pointer | `var p *T` | Zero value; never dereference without guard |
| Pointer to new zero value | `new(T)` | Returns `*T` |
| Auto-deref on structs | `p.Field` | Go sugar for `(*p).Field` |
| Slices, Maps, Channels | — | Already reference-like; no `&` needed |

---

## 1.9 Common Mistakes

```go
// ❌ Returning pointer to loop variable — all pointers point to the same address
ptrs := make([]*int, 3)
for i := 0; i < 3; i++ {
    ptrs[i] = &i  // BUG: all point to the same i
}

// ✅ Capture the loop variable explicitly
for i := 0; i < 3; i++ {
    v := i
    ptrs[i] = &v
}
```

---

*Next: [02 — Data Structures →](./02-data-structures.md)*
