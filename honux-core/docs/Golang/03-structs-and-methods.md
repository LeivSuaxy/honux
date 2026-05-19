# 03 — Structs & Methods in Go

> Structs are Go's primary mechanism for grouping data. Combined with methods and embedding, they provide a powerful, composition-based alternative to classical OOP inheritance.

---

## 3.1 Defining Structs

```go
type User struct {
    ID        int
    Name      string
    Email     string
    CreatedAt time.Time
}
```

### Struct Literals

```go
// Named fields (preferred — order-independent, readable)
u := User{
    ID:    1,
    Name:  "Alice",
    Email: "alice@example.com",
}

// Positional (fragile — breaks if fields are reordered; avoid for multi-field structs)
u2 := User{1, "Bob", "bob@example.com", time.Now()}

// Zero value — every field gets its type's zero value
var u3 User
fmt.Println(u3.Name) // ""
```

### Anonymous Structs

Useful for one-off data shapes, test fixtures, and JSON encoding.

```go
point := struct {
    X, Y int
}{X: 3, Y: 4}

// In tests
cases := []struct {
    input    string
    expected int
}{
    {"hello", 5},
    {"go", 2},
}
```

---

## 3.2 Struct Tags

Tags provide metadata for reflection-based libraries (JSON, SQL, validation, etc.).

```go
type Product struct {
    ID    int    `json:"id"    db:"product_id"`
    Name  string `json:"name"  db:"name"`
    Price float64 `json:"price,omitempty"`       // omit if zero
    Internal string `json:"-"`                    // always omit from JSON
}

p := Product{ID: 1, Name: "Widget", Price: 9.99}
b, _ := json.Marshal(p)
fmt.Println(string(b)) // {"id":1,"name":"Widget","price":9.99}
```

---

## 3.3 Methods

A method is a function with a **receiver** — a named type it belongs to.

```go
type Rectangle struct {
    Width, Height float64
}

// Value receiver — reads state
func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

// Value receiver — reads state
func (r Rectangle) Perimeter() float64 {
    return 2 * (r.Width + r.Height)
}

// Pointer receiver — mutates state
func (r *Rectangle) Scale(factor float64) {
    r.Width *= factor
    r.Height *= factor
}

func main() {
    rect := Rectangle{Width: 4, Height: 3}
    fmt.Println(rect.Area())       // 12
    rect.Scale(2)
    fmt.Println(rect.Width)        // 8
}
```

---

## 3.4 Constructor Functions

Go has no constructors, but the convention is a `New...` function that returns a fully initialized value (or pointer).

```go
type Server struct {
    host    string
    port    int
    timeout time.Duration
}

func NewServer(host string, port int) *Server {
    return &Server{
        host:    host,
        port:    port,
        timeout: 30 * time.Second, // sensible default
    }
}

// Functional options pattern — for complex optional configuration
type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func NewServerWithOptions(host string, port int, opts ...Option) *Server {
    s := &Server{host: host, port: port, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage
srv := NewServerWithOptions("localhost", 8080,
    WithTimeout(10*time.Second),
)
```

---

## 3.5 Embedding — Composition over Inheritance

Embedding a struct promotes its fields and methods to the outer struct.

```go
type Animal struct {
    Name string
}

func (a Animal) Speak() string {
    return a.Name + " makes a sound"
}

type Dog struct {
    Animal        // embedded — not a named field
    Breed string
}

func (d Dog) Speak() string {        // overrides Animal's Speak
    return d.Name + " barks"
}

func main() {
    d := Dog{
        Animal: Animal{Name: "Rex"},
        Breed:  "Labrador",
    }

    fmt.Println(d.Name)      // promoted from Animal
    fmt.Println(d.Speak())   // "Rex barks" — Dog's version
    fmt.Println(d.Animal.Speak()) // "Rex makes a sound" — explicit access
}
```

### Embedding Interfaces

You can embed interfaces in structs to declare that a struct satisfies an interface, or to compose interfaces.

```go
type Logger interface {
    Log(msg string)
}

type Service struct {
    Logger // embed the interface — provides the method set
    name string
}

// Any type implementing Logger can be injected
```

---

## 3.6 Struct Comparison

Structs are comparable with `==` if all their fields are comparable types.

```go
type Point struct{ X, Y int }

p1 := Point{1, 2}
p2 := Point{1, 2}
fmt.Println(p1 == p2) // true

// Structs containing slices, maps, or functions are NOT comparable
type Bad struct {
    Data []int
}
// Bad{} == Bad{} // compile error
```

---

## 3.7 Struct Copying

Since structs are value types, assignment copies the entire struct. Be aware of **shallow copy** when fields are pointers or slices.

```go
type Config struct {
    Ports []int
    Debug bool
}

original := Config{Ports: []int{8080, 8081}, Debug: true}
copied := original // shallow copy

copied.Ports[0] = 9999
fmt.Println(original.Ports[0]) // 9999 — shared backing array!

// Deep copy
func deepCopyConfig(c Config) Config {
    ports := make([]int, len(c.Ports))
    copy(ports, c.Ports)
    return Config{Ports: ports, Debug: c.Debug}
}
```

---

## 3.8 Method Sets & Interface Satisfaction

A type's **method set** determines which interfaces it satisfies.

```go
type Stringer interface {
    String() string
}

type Temperature struct {
    Celsius float64
}

func (t Temperature) String() string {
    return fmt.Sprintf("%.1f°C", t.Celsius)
}

// Temperature satisfies Stringer via value receiver
var s Stringer = Temperature{36.6}
fmt.Println(s.String()) // 36.6°C

// *Temperature also satisfies Stringer (pointer types include value method set)
var s2 Stringer = &Temperature{100}
fmt.Println(s2.String()) // 100.0°C
```

> **Rule:** A `*T` method set includes both pointer and value receivers. A `T` method set includes only value receivers.

---

## 3.9 Practical Example — Domain Model

```go
package main

import (
    "fmt"
    "time"
    "errors"
)

type Money struct {
    Amount   int64  // in cents
    Currency string
}

func (m Money) String() string {
    return fmt.Sprintf("%s %.2f", m.Currency, float64(m.Amount)/100)
}

func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, errors.New("currency mismatch")
    }
    return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

type OrderStatus string

const (
    StatusPending   OrderStatus = "pending"
    StatusPaid      OrderStatus = "paid"
    StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
    ID        string
    Items     []OrderItem
    Status    OrderStatus
    CreatedAt time.Time
}

type OrderItem struct {
    ProductID string
    Quantity  int
    UnitPrice Money
}

func (o *Order) Total() Money {
    total := Money{Currency: "USD"}
    for _, item := range o.Items {
        total.Amount += item.UnitPrice.Amount * int64(item.Quantity)
    }
    return total
}

func (o *Order) Cancel() error {
    if o.Status != StatusPending {
        return fmt.Errorf("cannot cancel order in status %q", o.Status)
    }
    o.Status = StatusCancelled
    return nil
}

func main() {
    order := &Order{
        ID:     "ord-001",
        Status: StatusPending,
        Items: []OrderItem{
            {ProductID: "p1", Quantity: 2, UnitPrice: Money{Amount: 999, Currency: "USD"}},
            {ProductID: "p2", Quantity: 1, UnitPrice: Money{Amount: 1499, Currency: "USD"}},
        },
        CreatedAt: time.Now(),
    }

    fmt.Println(order.Total()) // USD 34.97
    _ = order.Cancel()
    fmt.Println(order.Status)  // cancelled
}
```

---

*Next: [04 — Goroutines & Channels →](./04-goroutines-and-channels.md)*
