# Preconditions

Preconditions allow you to set up test data declaratively before running assertions.

## Installation

```go
import "testground"
```

## Interface

```go
type Precondition interface {
    Apply(ctx context.Context, t *testing.T) error
}
```

Any type implementing this interface can be used with `testground.Apply()`.

## Usage

```go
func TestExample(t *testing.T) {
    // ... setup container ...

    testground.Apply(t,
        precondition1,
        precondition2,
        precondition3,
    )

    // assertions...
}
```

If any precondition fails, the test stops immediately with a clear error message.

## Built-in Preconditions

### PostgreSQL: `Exec`

Execute SQL statements. Named arguments are optional:

```go
// Without arguments
container.Exec(`
    CREATE TABLE users (
        id   BIGSERIAL PRIMARY KEY,
        name TEXT NOT NULL
    )
`)

// With named arguments
container.Exec(
    `INSERT INTO users (name) VALUES (@name)`,
    pgx.NamedArgs{"name": "Alice"},
)
```

## Custom Preconditions

Create domain-specific preconditions by wrapping built-in ones:

```go
// testdata/preconditions.go

func InsertUser(pg *postgres.Container) func(name string) testground.Precondition {
    return func(name string) testground.Precondition {
        return pg.Exec(
            `INSERT INTO users (name) VALUES (@name)`,
            pgx.NamedArgs{"name": name},
        )
    }
}

func InsertOrder(pg *postgres.Container) func(userID int, amount int) testground.Precondition {
    return func(userID int, amount int) testground.Precondition {
        return pg.Exec(
            `INSERT INTO orders (user_id, amount) VALUES (@user_id, @amount)`,
            pgx.NamedArgs{"user_id": userID, "amount": amount},
        )
    }
}
```

Usage in tests:

```go
func TestOrders(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    pg, _ := postgres.New(ctx)
    s.Add(pg)

    // Create factories bound to container
    InsertUser := InsertUser(pg)
    InsertOrder := InsertOrder(pg)

    testground.Apply(t,
        pg.Exec(`CREATE TABLE users ...`),
        pg.Exec(`CREATE TABLE orders ...`),
        InsertUser("Alice"),
        InsertUser("Bob"),
        InsertOrder(1, 100),
        InsertOrder(2, 250),
    )

    // Test your code...
}
```

## Benefits

- **Declarative** — test setup reads like a specification
- **Composable** — combine simple preconditions into complex scenarios
- **Reusable** — define once, use across all tests
- **Type-safe** — no reflection, no magic, just functions
- **Fail-fast** — any error stops the test immediately