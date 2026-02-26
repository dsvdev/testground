# Suite

Suite manages container lifecycle and test hooks automatically.

## Installation

```go
import "github.com/dsvdev/testground/suite"
```

## Suite — for `Test*` functions

### Quick Start

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

### API

#### `New(t *testing.T) *Suite`

Creates a new Suite bound to the test. Automatically registers cleanup to terminate all containers.

#### `(*Suite) Add(c Managed)`

Registers a container to be managed by the Suite. The Suite takes ownership and calls `Terminate()` when the test completes.

```go
type Managed interface {
    Terminate(ctx context.Context) error
}
```

All testground containers implement this interface.

#### `(*Suite) Run(name string, fn func(t *testing.T))`

Runs a subtest with `BeforeEach`/`AfterEach` hooks.

#### Hooks

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

> **All hooks must be registered before the first `s.Run()` call.**
> Calling any hook-registration method after `s.Run()` has been called panics immediately with a descriptive message, e.g.:
> `panic: suite: BeforeAll must be called before the first Run`
>
> This catches the silent bug where a late-registered hook appears to succeed but never actually executes.

## MainSuite — for `TestMain`

`TestMain` receives `*testing.M` instead of `*testing.T`, so `Suite` cannot be used there. `MainSuite` fills this gap: it manages container lifecycle across the entire test binary.

### Quick Start

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    s := suite.NewMain(m)

    net, err := testground.NewNetwork(ctx)
    if err != nil {
        fmt.Printf("failed to create network: %v\n", err)
        os.Exit(1)
    }
    s.Add(net)

    pg, err := postgres.New(ctx,
        postgres.WithNetwork(net),
        postgres.WithNetworkAlias("postgres"),
    )
    if err != nil {
        fmt.Printf("failed to start postgres: %v\n", err)
        s.Cleanup()  // stop already-registered containers
        os.Exit(1)
    }
    s.Add(pg)

    // ... migrations, more services ...

    os.Exit(s.Run())  // m.Run() + Terminate in reverse order
}
```

### API

#### `NewMain(m *testing.M) *MainSuite`

Creates a new MainSuite bound to `*testing.M`.

#### `(*MainSuite) Add(c Managed)`

Registers a container to be stopped when `Run()` or `Cleanup()` is called.

#### `(*MainSuite) Run() int`

Calls `m.Run()`, then terminates all registered containers in reverse order. Returns the exit code to pass to `os.Exit`.

```go
os.Exit(s.Run())
```

#### `(*MainSuite) Cleanup()`

Terminates all registered containers in reverse order without running tests. Use this on early-exit error paths before `os.Exit(1)` to avoid leaving containers running in Docker.

```go
svcContainer, err = service.New(ctx, ...)
if err != nil {
    fmt.Printf("failed to start service: %v\n", err)
    s.Cleanup()  // stop net + pg that were already added
    os.Exit(1)
}
```

Errors from `Terminate` are printed via `fmt.Printf` — there is no `*testing.T` available in `TestMain`.

### Error Handling Pattern

Call `s.Cleanup()` before every `os.Exit(1)` that occurs after at least one `s.Add(...)`. Containers added before the failure are stopped; containers that never started are not in the list.

| Failure point | Added to suite | `s.Cleanup()` stops |
|---|---|---|
| `NewNetwork` fails | nothing | — (skip `Cleanup`) |
| `postgres.New` fails | `net` | `net` |
| migrations fail | `net`, `pg` | `pg` → `net` |
| `service.New` fails | `net`, `pg` | `pg` → `net` |
| Normal exit | `net`, `pg`, `svc` | `svc` → `pg` → `net` |

## Usage Patterns

### Shared Container (fast, tests share state)

Single container for all tests. Use `BeforeEach` to reset state between runs.

```go
func TestUserSuite(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    pg, _ := postgres.New(ctx)
    s.Add(pg)

    s.BeforeAll(func(ctx context.Context) {
        testground.Apply(t,
            pg.Exec(`CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL)`),
        )
    })

    s.BeforeEach(func(ctx context.Context) {
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

Each subtest gets its own container. No shared state, no cleanup needed between tests.

```go
func TestIsolated(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    s.Run("test A", func(t *testing.T) {
        inner := suite.New(t)
        pg, _ := postgres.New(ctx)
        inner.Add(pg)  // Terminated after "test A"
    })

    s.Run("test B", func(t *testing.T) {
        inner := suite.New(t)
        pg, _ := postgres.New(ctx)
        inner.Add(pg)  // Terminated after "test B"
    })
}
```

## Benefits

- **No manual cleanup** — `s.Add(pg)` replaces `t.Cleanup(func() { pg.Terminate() })`
- **Structured hooks** — `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`
- **Flexible isolation** — choose shared or isolated containers per test
- **Automatic termination** — containers cleaned up in reverse order
- **TestMain support** — `MainSuite` covers the full binary lifecycle with safe early-exit cleanup