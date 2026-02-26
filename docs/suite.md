# Suite

Suite manages container lifecycle and test hooks automatically.

## Installation

```go
import "testground/suite"
```

## Quick Start

```go
func TestUserSuite(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    pg, _ := postgres.New(ctx)
    s.Add(pg)  // No need for t.Cleanup — Suite handles it

    s.Run("create user", func(t *testing.T) {
        // test code
    })

    s.Run("get user", func(t *testing.T) {
        // test code
    })
}
```

## API

### `New(t *testing.T) *Suite`

Creates a new Suite bound to the test. Automatically registers cleanup to terminate all containers.

### `(*Suite) Add(c Managed)`

Registers a container to be managed by the Suite. The Suite takes ownership and calls `Terminate()` when the test completes.

```go
type Managed interface {
    Terminate(ctx context.Context) error
}
```

All testground containers implement this interface.

### `(*Suite) Run(name string, fn func(t *testing.T))`

Runs a subtest with `BeforeEach`/`AfterEach` hooks.

### Hooks

```go
s.BeforeAll(func(ctx context.Context) { ... })   // Once before first test
s.AfterAll(func(ctx context.Context) { ... })    // Once after all tests
s.BeforeEach(func(ctx context.Context) { ... })  // Before each s.Run()
s.AfterEach(func(ctx context.Context) { ... })   // After each s.Run()
```

Execution order:
```
BeforeAll → (BeforeEach → test → AfterEach)* → AfterAll
```

## Usage Patterns

### Shared Container (fast, tests share state)

Single container for all tests in a suite. Use `BeforeEach` to reset state.

```go
func TestUserSuite(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    pg, _ := postgres.New(ctx)
    s.Add(pg)

    s.BeforeAll(func(ctx context.Context) {
        // Run migrations once
        testground.Apply(t,
            pg.Exec(`CREATE TABLE users (id BIGSERIAL, name TEXT)`),
        )
    })

    s.BeforeEach(func(ctx context.Context) {
        // Clean tables before each test
        testground.Apply(t,
            pg.Exec(`TRUNCATE users RESTART IDENTITY`),
        )
    })

    s.Run("create user", func(t *testing.T) {
        // starts with empty table
    })

    s.Run("list users", func(t *testing.T) {
        // also starts with empty table
    })
}
```

### Isolated Containers (slower, full isolation)

Each test gets its own container. No shared state, no cleanup needed.

```go
func TestIsolated(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    s.Run("test A", func(t *testing.T) {
        inner := suite.New(t)
        pg, _ := postgres.New(ctx)
        inner.Add(pg)  // Terminated after "test A"

        // test with fresh database
    })

    s.Run("test B", func(t *testing.T) {
        inner := suite.New(t)
        pg, _ := postgres.New(ctx)
        inner.Add(pg)  // Terminated after "test B"

        // completely independent database
    })
}
```

## Benefits

- **No manual cleanup** — `s.Add(pg)` replaces `t.Cleanup(func() { pg.Terminate() })`
- **Structured hooks** — `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`
- **Flexible isolation** — choose shared or isolated containers per test
- **Automatic termination** — containers cleaned up in reverse order