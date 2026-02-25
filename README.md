# testground

[![CI](https://github.com/dsvdev/testground/actions/workflows/ci.yml/badge.svg)](https://github.com/dsvdev/testground/actions/workflows/ci.yml)

Mocks lie. **testground** lets you spin up a real environment — databases, caches, brokers — directly from your test code, so your integration tests run against the same infrastructure as production.

No Docker Compose. No shared environments. No cleanup.

## Services

- [PostgreSQL](docs/services/postgres.md)

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

    conn, _ := pgx.Connect(ctx, container.ConnectionString())
    defer conn.Close(ctx)

    // Your test code here
}
```

## Documentation

See [docs](docs/README.md) for full documentation.