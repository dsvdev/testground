# PostgreSQL

Spin up a real PostgreSQL container for integration tests.

## Installation

```go
import "testground/services/postgres"
```

## Quick Start

```go
func TestWithPostgres(t *testing.T) {
    ctx := context.Background()

    container, err := postgres.New(ctx)
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        container.Terminate(context.Background())
    })

    // Connect using pgx
    conn, err := pgx.Connect(ctx, container.ConnectionString())
    if err != nil {
        t.Fatal(err)
    }
    defer conn.Close(ctx)

    // Run queries
    var result int
    conn.QueryRow(ctx, "SELECT 1").Scan(&result)
}
```

## Options

All options are optional. Default values are applied if not specified.

| Option | Default | Description |
|--------|---------|-------------|
| `WithVersion(v)` | `"16"` | PostgreSQL version (Docker image tag) |
| `WithDatabase(d)` | `"test"` | Database name |
| `WithUser(u)` | `"test"` | Username |
| `WithPassword(p)` | `"test"` | Password |
| `WithPort(p)` | random | Host port (empty = random free port) |

### Examples

```go
// Default configuration
container, _ := postgres.New(ctx)

// Custom version
container, _ := postgres.New(ctx,
    postgres.WithVersion("15"),
)

// Full configuration
container, _ := postgres.New(ctx,
    postgres.WithVersion("16"),
    postgres.WithDatabase("myapp_test"),
    postgres.WithUser("admin"),
    postgres.WithPassword("secret"),
)

// Fixed port (useful for debugging)
container, _ := postgres.New(ctx,
    postgres.WithPort("5432"),
)
```

## API

### `New(ctx context.Context, opts ...Option) (*Container, error)`

Creates and starts a new PostgreSQL container.

### `(*Container) ConnectionString() string`

Returns the connection string in format:
```
postgres://user:password@host:port/database?sslmode=disable
```

### `(*Container) Conn(ctx context.Context) (*pgx.Conn, error)`

Returns a new pgx connection to the container. Caller is responsible for closing.

```go
conn, err := container.Conn(ctx)
if err != nil {
    t.Fatal(err)
}
defer conn.Close(ctx)
```

### `(*Container) Terminate(ctx context.Context) error`

Stops and removes the container. Always call this in `t.Cleanup()`.

### `(*Container) Exec(sql string, args pgx.NamedArgs) testground.Precondition`

Returns a [Precondition](../preconditions.md) that executes SQL when applied.

```go
testground.Apply(t,
    container.Exec(`CREATE TABLE users (id BIGSERIAL, name TEXT)`, nil),
    container.Exec(`INSERT INTO users (name) VALUES (@name)`, pgx.NamedArgs{"name": "Alice"}),
)
```

## Port Allocation

By default, testground uses a random free port for each container. This allows running multiple containers in parallel without port conflicts.

```go
// Two containers with different random ports
c1, _ := postgres.New(ctx)  // e.g., localhost:55004
c2, _ := postgres.New(ctx)  // e.g., localhost:55005
```

Use `WithPort()` only when you need a specific port (e.g., for debugging with external tools).