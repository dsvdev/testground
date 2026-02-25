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

    pg, err := postgres.New(ctx)
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        pg.Terminate(context.Background())
    })

    // Setup test data with preconditions
    testground.Apply(t,
        pg.Exec(`CREATE TABLE users (id BIGSERIAL, name TEXT)`, nil),
        pg.Exec(`INSERT INTO users (name) VALUES (@name)`, pgx.NamedArgs{"name": "Alice"}),
    )

    // Run your tests
    conn, _ := pg.Conn(ctx)
    defer conn.Close(ctx)

    var name string
    conn.QueryRow(ctx, "SELECT name FROM users WHERE id = 1").Scan(&name)
    // assert name == "Alice"
}
```

## Documentation

See [docs](docs/README.md) for full documentation.