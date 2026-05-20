# 09 — Event-Driven Architecture in Go

> This guide covers three practical levels of event-driven architecture in Go: an **in-process event bus** (no external dependencies), **PostgreSQL LISTEN/NOTIFY** (internal broker using your existing database), and **Apache Kafka** (external broker for large-scale systems).

---

## 9.1 Core Concepts

| Concept | Description |
|---|---|
| **Event** | An immutable fact that something happened: `OrderPlaced`, `UserRegistered` |
| **Publisher** | Emits events — doesn't know who listens |
| **Subscriber** | Reacts to events — doesn't know who publishes |
| **Broker** | Routes events between publishers and subscribers |
| **Dead Letter Queue** | Holds events that failed processing for later inspection |

### Event envelope

```go
// internal/event/event.go
package event

import (
    "time"
    "github.com/google/uuid"
)

type Event struct {
    ID          string          `json:"id"`
    Type        string          `json:"type"`        // e.g. "order.placed"
    AggregateID string          `json:"aggregate_id"` // e.g. order ID
    OccurredAt  time.Time       `json:"occurred_at"`
    Payload     json.RawMessage `json:"payload"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

func New(eventType, aggregateID string, payload any) (Event, error) {
    data, err := json.Marshal(payload)
    if err != nil {
        return Event{}, err
    }
    return Event{
        ID:          uuid.NewString(),
        Type:        eventType,
        AggregateID: aggregateID,
        OccurredAt:  time.Now().UTC(),
        Payload:     data,
    }, nil
}

// Typed payload extraction
func Unmarshal[T any](e Event) (T, error) {
    var v T
    err := json.Unmarshal(e.Payload, &v)
    return v, err
}
```

---

## 9.2 Level 1 — In-Process Event Bus

Suitable for a single-process application. Zero dependencies. Events are lost if the process restarts.

```go
// internal/event/bus.go
package event

import (
    "context"
    "log/slog"
    "sync"
)

type Handler func(ctx context.Context, e Event) error

type Bus struct {
    mu          sync.RWMutex
    subscribers map[string][]Handler
}

func NewBus() *Bus {
    return &Bus{subscribers: make(map[string][]Handler)}
}

// Subscribe registers a handler for a given event type.
func (b *Bus) Subscribe(eventType string, h Handler) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.subscribers[eventType] = append(b.subscribers[eventType], h)
}

// Publish dispatches the event to all registered handlers.
// Handlers run synchronously in the caller's goroutine.
func (b *Bus) Publish(ctx context.Context, e Event) error {
    b.mu.RLock()
    handlers := b.subscribers[e.Type]
    b.mu.RUnlock()

    for _, h := range handlers {
        if err := h(ctx, e); err != nil {
            slog.Error("event handler error",
                "event_type", e.Type,
                "event_id",   e.ID,
                "error",      err,
            )
        }
    }
    return nil
}

// PublishAsync dispatches the event in a new goroutine per handler.
func (b *Bus) PublishAsync(ctx context.Context, e Event) {
    b.mu.RLock()
    handlers := b.subscribers[e.Type]
    b.mu.RUnlock()

    for _, h := range handlers {
        h := h
        go func() {
            if err := h(ctx, e); err != nil {
                slog.Error("async event handler error",
                    "event_type", e.Type,
                    "event_id",   e.ID,
                    "error",      err,
                )
            }
        }()
    }
}
```

### Usage — In-Process Bus

```go
func main() {
    bus := event.NewBus()

    // Register handlers
    bus.Subscribe("order.placed", func(ctx context.Context, e event.Event) error {
        type OrderPlaced struct {
            OrderID    string  `json:"order_id"`
            CustomerID string  `json:"customer_id"`
            Amount     float64 `json:"amount"`
        }
        payload, err := event.Unmarshal[OrderPlaced](e)
        if err != nil {
            return err
        }
        slog.Info("sending confirmation email", "customer", payload.CustomerID)
        return nil
    })

    bus.Subscribe("order.placed", func(ctx context.Context, e event.Event) error {
        slog.Info("updating inventory", "event_id", e.ID)
        return nil
    })

    // Publish from service layer
    evt, _ := event.New("order.placed", "ord-001", map[string]any{
        "order_id":    "ord-001",
        "customer_id": "cust-42",
        "amount":      99.99,
    })
    bus.Publish(context.Background(), evt)
}
```

---

## 9.3 Level 2 — PostgreSQL LISTEN / NOTIFY

Uses your existing PostgreSQL database as a message broker. Durable, reliable, no extra infrastructure.

### Database schema

```sql
-- Outbox table — transactionally reliable event storage
CREATE TABLE outbox_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type   TEXT        NOT NULL,
    aggregate_id TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    error        TEXT
);

CREATE INDEX idx_outbox_unprocessed ON outbox_events(created_at)
    WHERE processed_at IS NULL;
```

### Publisher — Transactional Outbox Pattern

The publisher writes the event inside the same database transaction as the business operation. This guarantees the event is never lost even if the process crashes.

```go
// internal/event/pgpublisher.go
package event

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
)

type PgPublisher struct {
    db *sql.DB
}

func NewPgPublisher(db *sql.DB) *PgPublisher {
    return &PgPublisher{db: db}
}

// PublishInTx writes the event to the outbox table within an existing transaction.
func (p *PgPublisher) PublishInTx(ctx context.Context, tx *sql.Tx, e Event) error {
    payload, err := json.Marshal(e.Payload)
    if err != nil {
        return fmt.Errorf("PublishInTx marshal: %w", err)
    }

    _, err = tx.ExecContext(ctx, `
        INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `, e.ID, e.Type, e.AggregateID, payload, e.OccurredAt)
    return err
}
```

### Outbox Relay — Forward Events via NOTIFY

A background worker polls for unprocessed events and forwards them via `NOTIFY`:

```go
// internal/event/relay.go
package event

import (
    "context"
    "database/sql"
    "encoding/json"
    "log/slog"
    "time"
)

type Relay struct {
    db       *sql.DB
    interval time.Duration
}

func NewRelay(db *sql.DB) *Relay {
    return &Relay{db: db, interval: 500 * time.Millisecond}
}

func (r *Relay) Start(ctx context.Context) {
    ticker := time.NewTicker(r.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := r.relay(ctx); err != nil {
                slog.Error("relay error", "error", err)
            }
        }
    }
}

func (r *Relay) relay(ctx context.Context) error {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, event_type, aggregate_id, payload
        FROM outbox_events
        WHERE processed_at IS NULL
        ORDER BY created_at
        LIMIT 100
        FOR UPDATE SKIP LOCKED
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id, eventType, aggregateID string
        var payload []byte
        if err := rows.Scan(&id, &eventType, &aggregateID, &payload); err != nil {
            return err
        }

        // NOTIFY channel with the event as JSON
        notifyPayload, _ := json.Marshal(map[string]string{
            "id":           id,
            "event_type":   eventType,
            "aggregate_id": aggregateID,
        })

        _, err := r.db.ExecContext(ctx,
            `SELECT pg_notify($1, $2)`,
            "events:"+eventType, string(notifyPayload),
        )
        if err != nil {
            slog.Error("pg_notify error", "id", id, "error", err)
            continue
        }

        // Mark as processed
        r.db.ExecContext(ctx,
            `UPDATE outbox_events SET processed_at = NOW() WHERE id = $1`, id)
    }
    return rows.Err()
}
```

### Subscriber — LISTEN with pgx

```go
// internal/event/pglistener.go
package event

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

type PgListener struct {
    pool     *pgxpool.Pool
    handlers map[string][]Handler
}

func NewPgListener(pool *pgxpool.Pool) *PgListener {
    return &PgListener{
        pool:     pool,
        handlers: make(map[string][]Handler),
    }
}

func (l *PgListener) Subscribe(eventType string, h Handler) {
    l.handlers[eventType] = append(l.handlers[eventType], h)
}

func (l *PgListener) Listen(ctx context.Context) error {
    conn, err := l.pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    pgConn := conn.Hijack()
    defer pgConn.Close(ctx)

    for channel := range l.handlers {
        _, err := pgConn.Exec(ctx, "LISTEN "+channel)
        if err != nil {
            return fmt.Errorf("LISTEN %s: %w", channel, err)
        }
        slog.Info("pg listener subscribed", "channel", channel)
    }

    for {
        notification, err := pgConn.WaitForNotification(ctx)
        if err != nil {
            if ctx.Err() != nil {
                return nil // context cancelled — clean shutdown
            }
            return fmt.Errorf("WaitForNotification: %w", err)
        }
        go l.dispatch(ctx, notification)
    }
}

func (l *PgListener) dispatch(ctx context.Context, n *pgconn.Notification) {
    // channel name is "events:<event_type>"
    eventType := strings.TrimPrefix(n.Channel, "events:")
    handlers := l.handlers[eventType]

    var meta map[string]string
    json.Unmarshal([]byte(n.Payload), &meta)

    e := Event{
        ID:          meta["id"],
        Type:        eventType,
        AggregateID: meta["aggregate_id"],
    }

    for _, h := range handlers {
        if err := h(ctx, e); err != nil {
            slog.Error("pg listener handler error",
                "channel", n.Channel,
                "event_id", e.ID,
                "error", err,
            )
        }
    }
}
```

---

## 9.4 Level 3 — Apache Kafka

For high-throughput, persistent, replay-capable event streaming.

```bash
go get github.com/twmb/franz-go/pkg/kgo
```

### Producer

```go
// internal/event/kafkaproducer.go
package event

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/twmb/franz-go/pkg/kgo"
)

type KafkaProducer struct {
    client *kgo.Client
}

func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
    client, err := kgo.NewClient(
        kgo.SeedBrokers(brokers...),
        kgo.RequiredAcks(kgo.AllISRAcks()),  // wait for all replicas
        kgo.ProducerLinger(5 * time.Millisecond), // micro-batch
    )
    if err != nil {
        return nil, fmt.Errorf("kafka producer: %w", err)
    }
    return &KafkaProducer{client: client}, nil
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, e Event) error {
    payload, err := json.Marshal(e)
    if err != nil {
        return fmt.Errorf("kafka marshal: %w", err)
    }

    record := &kgo.Record{
        Topic: topic,
        Key:   []byte(e.AggregateID), // key = aggregate ID ensures ordering per entity
        Value: payload,
        Headers: []kgo.RecordHeader{
            {Key: "event_type", Value: []byte(e.Type)},
            {Key: "event_id",   Value: []byte(e.ID)},
        },
    }

    results := p.client.ProduceSync(ctx, record)
    return results.FirstErr()
}

func (p *KafkaProducer) Close() { p.client.Close() }
```

### Consumer

```go
// internal/event/kafkaconsumer.go
package event

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/twmb/franz-go/pkg/kgo"
)

type KafkaConsumer struct {
    client   *kgo.Client
    handlers map[string]Handler
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string) (*KafkaConsumer, error) {
    client, err := kgo.NewClient(
        kgo.SeedBrokers(brokers...),
        kgo.ConsumerGroup(groupID),
        kgo.ConsumeTopics(topics...),
        kgo.DisableAutoCommit(),    // commit manually for at-least-once delivery
    )
    if err != nil {
        return nil, err
    }
    return &KafkaConsumer{
        client:   client,
        handlers: make(map[string]Handler),
    }, nil
}

func (c *KafkaConsumer) Register(eventType string, h Handler) {
    c.handlers[eventType] = h
}

func (c *KafkaConsumer) Consume(ctx context.Context) error {
    for {
        fetches := c.client.PollFetches(ctx)
        if fetches.IsClientClosed() || ctx.Err() != nil {
            return nil
        }
        if errs := fetches.Errors(); len(errs) > 0 {
            for _, e := range errs {
                slog.Error("kafka fetch error", "error", e.Err)
            }
        }

        fetches.EachRecord(func(record *kgo.Record) {
            var e Event
            if err := json.Unmarshal(record.Value, &e); err != nil {
                slog.Error("kafka unmarshal error", "error", err)
                return
            }

            handler, ok := c.handlers[e.Type]
            if !ok {
                slog.Warn("no handler for event type", "type", e.Type)
                return
            }

            if err := handler(ctx, e); err != nil {
                slog.Error("kafka handler error",
                    "event_type", e.Type,
                    "event_id",   e.ID,
                    "partition",  record.Partition,
                    "offset",     record.Offset,
                    "error",      err,
                )
                // Don't commit — message will be re-delivered
                return
            }
        })

        // Commit only after successful processing
        if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
            slog.Error("kafka commit error", "error", err)
        }
    }
}

func (c *KafkaConsumer) Close() { c.client.Close() }
```

---

## 9.5 Wiring It All Together

```go
// cmd/worker/main.go — background event processor
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    consumer, err := event.NewKafkaConsumer(
        []string{os.Getenv("KAFKA_BROKER")},
        "order-service",
        []string{"order.placed", "order.cancelled"},
    )
    if err != nil {
        slog.Error("kafka consumer init", "error", err)
        os.Exit(1)
    }
    defer consumer.Close()

    // Wire handlers
    consumer.Register("order.placed",    handleOrderPlaced)
    consumer.Register("order.cancelled", handleOrderCancelled)

    slog.Info("consumer starting...")
    if err := consumer.Consume(ctx); err != nil {
        slog.Error("consumer error", "error", err)
    }
    slog.Info("consumer stopped")
}

func handleOrderPlaced(ctx context.Context, e event.Event) error {
    type Payload struct {
        OrderID    string  `json:"order_id"`
        CustomerID string  `json:"customer_id"`
        Amount     float64 `json:"amount"`
    }
    p, err := event.Unmarshal[Payload](e)
    if err != nil {
        return err
    }
    slog.Info("processing order placed",
        "order_id",    p.OrderID,
        "customer_id", p.CustomerID,
        "amount",      p.Amount,
    )
    // ... send email, update analytics, etc.
    return nil
}

func handleOrderCancelled(ctx context.Context, e event.Event) error {
    slog.Info("processing order cancelled", "event_id", e.ID)
    return nil
}
```

---

## 9.6 Dead Letter Queue

```go
// internal/event/dlq.go
package event

type DLQ struct {
    db *sql.DB
}

func (d *DLQ) Store(ctx context.Context, e Event, processingError error) error {
    payload, _ := json.Marshal(e)
    _, err := d.db.ExecContext(ctx, `
        INSERT INTO dead_letter_queue (event_id, event_type, payload, error, failed_at)
        VALUES ($1, $2, $3, $4, NOW())
    `, e.ID, e.Type, payload, processingError.Error())
    return err
}
```

```sql
CREATE TABLE dead_letter_queue (
    id         BIGSERIAL   PRIMARY KEY,
    event_id   TEXT        NOT NULL,
    event_type TEXT        NOT NULL,
    payload    JSONB       NOT NULL,
    error      TEXT        NOT NULL,
    failed_at  TIMESTAMPTZ NOT NULL,
    retried_at TIMESTAMPTZ,
    resolved   BOOLEAN     NOT NULL DEFAULT FALSE
);
```

---

## 9.7 Choosing the Right Approach

| Approach | Durability | Throughput | Dependencies | Best For |
|---|---|---|---|---|
| In-process Bus | None (in-memory) | Highest | None | Single-process, non-critical side effects |
| PG LISTEN/NOTIFY | Yes (outbox) | Medium | Existing PostgreSQL | Small-medium apps, strong consistency needed |
| Kafka | Yes (log) | Very high | Kafka cluster | High throughput, event replay, microservices |
| Redis Streams | Yes (configurable) | High | Redis | Real-time, moderate durability needs |
| NATS | Configurable | Very high | NATS server | Low-latency, simple setup |

---

## 9.8 Key Patterns Summary

```
Transactional Outbox
  ├── Write event to outbox IN the same DB transaction as the business change
  ├── Background relay reads outbox and forwards to broker
  └── Guarantees: at-least-once delivery, no events lost on crash

Consumer Group
  ├── Multiple consumer instances share partitions
  ├── Each event processed by exactly one instance in the group
  └── Scale consumers = scale throughput

Dead Letter Queue
  ├── Failed events stored separately for inspection
  ├── Can be replayed after fixing the handler bug
  └── Prevents one bad message from blocking the queue
```

---

*End of Go Developer Documentation*

---

## Index

| File | Topic |
|---|---|
| [01-pointers-and-references.md](./01-pointers-and-references.md) | Pointers, references, escape analysis |
| [02-data-structures.md](./02-data-structures.md) | Slice, map, stack, queue, heap, set |
| [03-structs-and-methods.md](./03-structs-and-methods.md) | Structs, embedding, constructor patterns |
| [04-goroutines-and-channels.md](./04-goroutines-and-channels.md) | Goroutines, channels, select, context, sync |
| [05-interfaces.md](./05-interfaces.md) | Interface design, type assertions, generics |
| [06-http-handlers-and-endpoints.md](./06-http-handlers-and-endpoints.md) | ServeMux, layered architecture, DI |
| [07-middlewares-and-http-best-practices.md](./07-middlewares-and-http-best-practices.md) | Middleware chain, auth, rate limit, security |
| [08-postgresql.md](./08-postgresql.md) | pgx, queries, transactions, migrations |
| [09-event-driven-architecture.md](./09-event-driven-architecture.md) | In-process bus, PG LISTEN/NOTIFY, Kafka |
