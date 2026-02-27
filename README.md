# testground

[![CI](https://github.com/dsvdev/testground/actions/workflows/ci.yml/badge.svg)](https://github.com/dsvdev/testground/actions/workflows/ci.yml)

Mocks lie. **testground** lets you spin up a real environment — databases, caches, brokers — directly from your test code, so your integration tests run against the same infrastructure as production.

No Docker Compose. No shared environments. No cleanup.

## Services

- [PostgreSQL](docs/services/postgres.md)

## Client

- [HTTP](docs/client/httpclient.md) — fluent HTTP client with assertions

## Quick Start

```go
// Custom precondition for your domain
func InsertUser(pg *postgres.Container) func(name string) testground.Precondition {
    return func(name string) testground.Precondition {
        return pg.Exec(
            `INSERT INTO users (name) VALUES (@name)`,
            pgx.NamedArgs{"name": name},
        )
    }
}

func TestUserSuite(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    pg, _ := postgres.New(ctx)
    s.Add(pg)

    // Run migrations once
    s.BeforeAll(func(ctx context.Context) {
        testground.Apply(t, pg.Exec(`CREATE TABLE users (id BIGSERIAL, name TEXT)`))
    })

    // Create precondition factory
    insertUser := InsertUser(pg)

    s.Run("create and fetch user", func(t *testing.T) {
        testground.Apply(t, insertUser("Alice"))

        conn, _ := pg.Conn(ctx)
        defer conn.Close(ctx)

        var name string
        conn.QueryRow(ctx, "SELECT name FROM users WHERE id = 1").Scan(&name)
        // assert name == "Alice"
    })
}
```

## HTTP Client

```go
import "testground/client/httpclient"

func TestAPI(t *testing.T) {
    ctx := context.Background()
    client := httpclient.New(httpclient.WithBaseURL("http://localhost:8080"))

    resp, err := client.Get(ctx, "/users/1")
    require.NoError(t, err)

    var user User
    resp.AssertOK(t).AssertJSON(t, &user)
}
```

See [docs/client/httpclient.md](docs/client/httpclient.md) for full API reference.

## Test Data

```go
import "github.com/dsvdev/testground/faker"

name := faker.RandomString(10)
age  := faker.RandomInt(18, 65)
id   := faker.RandomUUID()
```

See [faker](docs/faker.md) for full API reference.

## Documentation

See [docs](docs/README.md) for full documentation.