# 05 — Interfaces in Go

> Interfaces in Go are **implicit** — a type satisfies an interface simply by implementing its methods. No `implements` keyword. This makes Go interfaces lightweight, decoupled, and composable.

---

## 5.1 Defining an Interface

```go
type Stringer interface {
    String() string
}
```

Any type that has a `String() string` method automatically satisfies `Stringer`. No declaration needed.

```go
type Temperature struct {
    Celsius float64
}

func (t Temperature) String() string {
    return fmt.Sprintf("%.1f°C", t.Celsius)
}

var s Stringer = Temperature{36.6} // Temperature satisfies Stringer implicitly
fmt.Println(s.String())            // 36.6°C
```

---

## 5.2 Interface Values

An interface value holds two things: a **concrete type** and a **concrete value**. Both together form the dynamic dispatch mechanism.

```go
var s Stringer // nil interface: type=nil, value=nil

s = Temperature{100}
// s's dynamic type: Temperature
// s's dynamic value: {100}

fmt.Printf("type: %T  value: %v\n", s, s)
// type: main.Temperature  value: 100.0°C
```

### Nil Interface vs Interface Holding Nil Pointer

This is a famous Go gotcha:

```go
type MyError struct{ msg string }
func (e *MyError) Error() string { return e.msg }

func getError(fail bool) error {
    var err *MyError // typed nil pointer
    if fail {
        err = &MyError{"something went wrong"}
    }
    return err // ALWAYS non-nil interface! (type=*MyError, value=nil)
}

e := getError(false)
if e != nil { // this is TRUE even though err was nil
    fmt.Println("unexpected:", e) // executes
}

// ✅ Correct: return the untyped nil
func getErrorFixed(fail bool) error {
    if fail {
        return &MyError{"something went wrong"}
    }
    return nil // untyped nil — interface is nil
}
```

---

## 5.3 Interface Composition

Interfaces can embed other interfaces.

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Composed interface
type ReadWriter interface {
    Reader
    Writer
}

// A type satisfies ReadWriter only if it implements both Read and Write
```

Standard library examples of composed interfaces: `io.ReadWriteCloser`, `http.ResponseWriter`, `io.ReadSeeker`.

---

## 5.4 Small, Focused Interfaces

Go favors **small interfaces**. The standard library's most powerful interfaces are tiny.

```go
// From the standard library
type error interface {
    Error() string
}

type io.Reader interface {
    Read(p []byte) (n int, err error)
}

type io.Writer interface {
    Write(p []byte) (n int, err error)
}

type fmt.Stringer interface {
    String() string
}
```

Design your own interfaces to be just as focused. A 10-method interface is usually a design smell.

---

## 5.5 Practical Design — Dependency Injection via Interfaces

```go
// Define the interface where it's consumed, not where it's implemented.

// In package "order":
type PaymentProcessor interface {
    Charge(amount int64, currency, token string) error
}

type OrderService struct {
    payments PaymentProcessor
    // ...
}

func NewOrderService(p PaymentProcessor) *OrderService {
    return &OrderService{payments: p}
}

func (s *OrderService) Checkout(amount int64, token string) error {
    return s.payments.Charge(amount, "USD", token)
}

// In package "stripe":
type StripeProcessor struct{ apiKey string }

func (p *StripeProcessor) Charge(amount int64, currency, token string) error {
    fmt.Printf("stripe charge: %d %s token=%s\n", amount, currency, token)
    return nil
}

// In package "main" — wire together
func main() {
    stripe := &StripeProcessor{apiKey: "sk_test_..."}
    svc := NewOrderService(stripe)
    _ = svc.Checkout(999, "tok_test")
}
```

Now `OrderService` has zero knowledge of Stripe. You can swap in a `MockProcessor` for tests.

---

## 5.6 Type Assertions

A type assertion extracts the concrete value from an interface.

```go
var i interface{} = "hello"

// Panics if i doesn't hold a string
s := i.(string)
fmt.Println(s) // "hello"

// Safe (two-value) form — never panics
s, ok := i.(string)
if ok {
    fmt.Println("string:", s)
}

n, ok := i.(int)
fmt.Println(n, ok) // 0, false
```

---

## 5.7 Type Switches

A type switch dispatches on the dynamic type of an interface value.

```go
func describe(i interface{}) string {
    switch v := i.(type) {
    case int:
        return fmt.Sprintf("int: %d", v)
    case string:
        return fmt.Sprintf("string: %q (len=%d)", v, len(v))
    case bool:
        return fmt.Sprintf("bool: %t", v)
    case []int:
        return fmt.Sprintf("[]int of length %d", len(v))
    case nil:
        return "nil"
    default:
        return fmt.Sprintf("unknown type: %T", v)
    }
}
```

---

## 5.8 The Empty Interface: `any` (alias for `interface{}`)

`any` accepts any value. Use it sparingly — it discards type safety.

```go
func printAnything(v any) {
    fmt.Println(v)
}

// Common in generic containers (before generics were added in Go 1.18)
type Container struct {
    items []any
}
```

Since Go 1.18, prefer **generics** over `any` when type safety matters.

---

## 5.9 Interfaces & Generics (Go 1.18+)

Type constraints in generics are interfaces.

```go
type Number interface {
    int | int64 | float64
}

func Sum[T Number](nums []T) T {
    var total T
    for _, n := range nums {
        total += n
    }
    return total
}

fmt.Println(Sum([]int{1, 2, 3}))       // 6
fmt.Println(Sum([]float64{1.1, 2.2})) // 3.3
```

---

## 5.10 Standard Library Interfaces to Know

```go
// error — implement for custom errors
type ValidationError struct {
    Field   string
    Message string
}
func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %q: %s", e.Field, e.Message)
}

// fmt.Stringer — controls default fmt output
type Color int
const (Red Color = iota; Green; Blue)
func (c Color) String() string {
    return [...]string{"Red", "Green", "Blue"}[c]
}

// io.Reader / io.Writer — compose streaming I/O
func processStream(r io.Reader) ([]byte, error) {
    return io.ReadAll(r)
}
// Works with: os.File, bytes.Buffer, http.Response.Body, strings.Reader, etc.

// sort.Interface — custom sort
type ByLength []string
func (s ByLength) Len() int           { return len(s) }
func (s ByLength) Less(i, j int) bool { return len(s[i]) < len(s[j]) }
func (s ByLength) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

words := []string{"banana", "kiwi", "fig"}
sort.Sort(ByLength(words))
fmt.Println(words) // [fig kiwi banana]
```

---

## 5.11 Interface Design Rules

| Rule | Explanation |
|---|---|
| Accept interfaces, return structs | Functions take interfaces for flexibility; callers get concrete types |
| Define interfaces at the point of use | Don't define them in the package that implements them |
| Keep interfaces small | 1–3 methods is ideal |
| Don't pre-emptively abstract | Extract an interface when you have 2+ implementations |
| Prefer composition | Embed interfaces to build larger ones |

---

*Next: [06 — HTTP Handlers & Endpoints →](./06-http-handlers-and-endpoints.md)*
