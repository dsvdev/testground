# Network

Shared Docker bridge network for container-to-container communication in integration tests.

## Installation

```go
import "testground"
```

## Quick Start

Create a network so containers can reach each other using stable aliases instead of dynamic host ports:

```go
func TestWithNetwork(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    net, err := testground.NewNetwork(ctx)
    if err != nil {
        t.Fatal(err)
    }
    s.Add(net)

    pg, err := postgres.New(ctx,
        postgres.WithNetwork(net),
        postgres.WithNetworkAlias("postgres"),
    )
    s.Add(pg)

    svc, err := service.New(ctx,
        service.WithBuildContext("."),
        service.WithDockerfile("Dockerfile"),
        service.WithNetwork(net),
        service.WithEnv("DATABASE_URL", pg.NetworkConnectionString()),
        service.WithPort("8080"),
    )
    s.Add(svc)
}
```

## API

### `NewNetwork(ctx context.Context) (*Network, error)`

Creates a new Docker bridge network with a random name.

### `(*Network) Name() string`

Returns the Docker network name. Used internally by `WithNetwork` options on containers.

### `(*Network) Terminate(ctx context.Context) error`

Removes the Docker network. Prefer using [Suite](suite.md) instead of calling manually:

```go
s := suite.New(t)
s.Add(net)  // Terminate called automatically
```

## Why Use a Network?

Without a network, containers communicate only via host port mappings (e.g. `localhost:55004`). A host port is not reachable from inside another container, so when your service container needs to connect to a database container at startup, you must use a Docker network.

With a network and aliases, the connection string uses a stable hostname:

```
postgres://test:test@postgres:5432/test?sslmode=disable
```

Use `(*postgres.Container).NetworkConnectionString()` to get this form.

## Container Support

| Container | Option | Alias option |
|-----------|--------|--------------|
| `postgres.Container` | `postgres.WithNetwork(net)` | `postgres.WithNetworkAlias(alias)` |
| `service.Container` | `service.WithNetwork(net)` | — |