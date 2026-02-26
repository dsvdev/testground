# Service Container

Run your application under test as a Docker container built from a local Dockerfile.

## Installation

```go
import "testground/service"
```

## Quick Start

```go
func TestMyService(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    svc, err := service.New(ctx,
        service.WithBuildContext("."),
        service.WithDockerfile("Dockerfile"),
        service.WithPort("8080"),
    )
    if err != nil {
        t.Fatal(err)
    }
    s.Add(svc)

    client := http.New(http.WithBaseURL(svc.URL()))
    // make requests...
}
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithBuildContext(path)` | `"."` | Docker build context directory |
| `WithDockerfile(path)` | — | Dockerfile path relative to build context |
| `WithPort(port)` | — | Container port to expose (e.g. `"8080"`) |
| `WithEnv(key, value)` | — | Set an environment variable inside the container |
| `WithNetwork(n)` | — | Attach container to a Docker network |
| `WithWaitFor(s)` | `ForListeningPort("8080/tcp")` | Custom readiness wait strategy |

### Examples

```go
// Minimal — build from current directory
svc, _ := service.New(ctx,
    service.WithBuildContext("."),
    service.WithDockerfile("Dockerfile"),
    service.WithPort("8080"),
)

// Build context is module root, Dockerfile is nested
svc, _ := service.New(ctx,
    service.WithBuildContext("../../"),
    service.WithDockerfile("example/myapp/Dockerfile"),
    service.WithPort("8080"),
)

// With environment variables
svc, _ := service.New(ctx,
    service.WithBuildContext("."),
    service.WithDockerfile("Dockerfile"),
    service.WithPort("8080"),
    service.WithEnv("LOG_LEVEL", "debug"),
    service.WithEnv("DATABASE_URL", pg.NetworkConnectionString()),
)
```

## API

### `New(ctx context.Context, opts ...Option) (*Container, error)`

Builds and starts a new container from a local Dockerfile. Blocks until the container is ready (default: port 8080 is listening).

### `(*Container) URL() string`

Returns the base URL for the container: `http://host:mapped-port`. Use this with the HTTP client:

```go
client := http.New(http.WithBaseURL(svc.URL()))
```

### `(*Container) Port() string`

Returns the mapped host port as a string.

### `(*Container) Terminate(ctx context.Context) error`

Stops and removes the container. Prefer using [Suite](suite.md) instead of calling manually.

## Integration with Network

Combine with `Network` and `postgres.Container` for a complete integration test setup:

```go
func TestMain(m *testing.M) {
    ctx := context.Background()

    net, _ := testground.NewNetwork(ctx)

    pg, _ := postgres.New(ctx,
        postgres.WithNetwork(net),
        postgres.WithNetworkAlias("postgres"),
    )

    svc, _ := service.New(ctx,
        service.WithBuildContext("../../"),
        service.WithDockerfile("example/myapp/Dockerfile"),
        service.WithNetwork(net),
        service.WithEnv("DATABASE_URL", pg.NetworkConnectionString()),
        service.WithPort("8080"),
    )

    client = http.New(http.WithBaseURL(svc.URL()))

    code := m.Run()

    svc.Terminate(ctx)
    pg.Terminate(ctx)
    net.Terminate(ctx)

    os.Exit(code)
}
```

See the [full example](../example/simple_backend/integration_test.go) for a working integration test.

## Dockerfile Tips

Use a multi-stage build to keep the final image small. When the build context is the module root, reference the entrypoint by its full path within the module:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app_binary ./path/to/cmd

FROM alpine:latest
COPY --from=builder /app_binary /app_binary
EXPOSE 8080
CMD ["/app_binary"]
```