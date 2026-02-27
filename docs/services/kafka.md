# Kafka

Kafka service for integration testing. Spins up a Zookeeper + Kafka pair inside
a private Docker network. From the test code you see a single `kafka.Container`
with a simple API.

## Installation

```go
import "github.com/dsvdev/testground/services/kafka"
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithVersion(v)` | `"7.6.1"` | Docker image version for both cp-zookeeper and cp-kafka |
| `WithNetwork(n)` | — | Attach Kafka to an external network (for container-to-container use) |
| `WithNetworkAlias(alias)` | `"kafka"` | Alias for Kafka inside the external network |

## API

```go
// External address — use from test code running on the host.
kc.BootstrapServers() string

// Internal address — use from other containers inside the shared network.
kc.NetworkBootstrapServers() string

kc.Terminate(ctx context.Context) error
```

## Preconditions

```go
// Create a topic (no-op if it already exists).
kc.CreateTopic("events", kafka.WithPartitions(1))

// Publish one message.
kc.Publish("events", []byte(`{"id": 1}`))
```

`TopicOption`:

| Option | Default | Description |
|--------|---------|-------------|
| `WithPartitions(n)` | `1` | Number of partitions |
| `WithReplicationFactor(n)` | `1` | Replication factor |

## Assertions

All assertions read the topic from the beginning with a 30-second timeout
and call `t.Fatal` on failure.

```go
// Exact message count.
kc.AssertMessageCount(t, "events", 3)

// At least one message with this exact value.
kc.AssertHasMessage(t, "events", []byte(`{"id": 2}`))

// Exactly wantCount messages whose value contains substr.
kc.AssertHasMessageContaining(t, "events", `"id": 1`, 2)
```

## Standalone example

```go
func TestKafka(t *testing.T) {
    ctx := context.Background()

    kc, err := kafka.New(ctx)
    if err != nil {
        t.Fatal(err)
    }
    defer kc.Terminate(ctx)

    testground.Apply(t,
        kc.CreateTopic("events", kafka.WithPartitions(1)),
        kc.Publish("events", []byte(`{"id": 1}`)),
        kc.Publish("events", []byte(`{"id": 1}`)),
        kc.Publish("events", []byte(`{"id": 2}`)),
    )

    kc.AssertMessageCount(t, "events", 3)
    kc.AssertHasMessage(t, "events", []byte(`{"id": 2}`))
    kc.AssertHasMessageContaining(t, "events", `"id": 1`, 2)
}
```

## Example with external network

Use this when a containerised service also needs to publish or consume messages:

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    s := suite.NewMain(m)

    net, err := testground.NewNetwork(ctx)
    if err != nil {
        log.Fatal(err)
    }
    s.Add(net)

    kc, err := kafka.New(ctx,
        kafka.WithNetwork(net),
        kafka.WithNetworkAlias("kafka"),
    )
    if err != nil {
        log.Fatal(err)
    }
    s.Add(kc)

    // Your service connects to kc.NetworkBootstrapServers() ("kafka:9092")
    // from inside the shared network.
    // Test code connects to kc.BootstrapServers() from the host.

    os.Exit(s.Run())
}
```

## Notes

- Zookeeper and Kafka always share the same image version.
- The external listener uses a randomly allocated host port chosen at startup.
- `Terminate` stops Kafka first, then Zookeeper, then the internal network.