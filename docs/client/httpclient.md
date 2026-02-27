# HTTP Client

HTTP client designed for integration testing with fluent assertions.

## Installation

```go
import "testground/client/httpclient"
```

## Quick Start

```go
func TestAPI(t *testing.T) {
    ctx := context.Background()
    client := httpclient.New(httpclient.WithBaseURL("http://localhost:8080"))

    resp, err := client.Get(ctx, "/users/1")
    require.NoError(t, err)

    var user User
    resp.AssertOK(t).AssertJSON(t, &user)
}
```

## Client Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `""` | Base URL for all requests |
| `WithTimeout(d)` | `30s` | Request timeout |
| `WithHeader(k, v)` | — | Add global header |
| `WithBearerToken(token)` | — | Add `Authorization: Bearer <token>` |

### Examples

```go
// Basic client
client := httpclient.New(httpclient.WithBaseURL("http://localhost:8080"))

// With authentication
client := httpclient.New(
    httpclient.WithBaseURL("http://localhost:8080"),
    httpclient.WithBearerToken("my-jwt-token"),
)

// With custom headers
client := httpclient.New(
    httpclient.WithBaseURL("http://localhost:8080"),
    httpclient.WithHeader("X-API-Key", "secret"),
    httpclient.WithTimeout(10 * time.Second),
)
```

## Request Methods

```go
client.Get(ctx context.Context, path string, opts ...RequestOption) (*Response, error)
client.Post(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error)
client.Put(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error)
client.Patch(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error)
client.Delete(ctx context.Context, path string, opts ...RequestOption) (*Response, error)
```

All methods:
- Accept `context.Context` for cancellation and timeouts
- Serialize `body` to JSON automatically
- Merge global headers with request-specific headers
- Return `(*Response, error)` — never panic

## Request Options

Override settings for a single request:

```go
ctx := context.Background()

// Add query parameters
client.Get(ctx, "/users",
    httpclient.WithQueryParam("page", "1"),
    httpclient.WithQueryParam("limit", "10"),
)
// → GET /users?page=1&limit=10

// Add request-specific header
client.Get(ctx, "/resource",
    httpclient.WithRequestHeader("X-Custom", "value"),
)

// With timeout context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
client.Get(ctx, "/slow-endpoint")
```

## Response

### Reading the Body

```go
// As struct (JSON)
var user User
resp.JSON(&user)

// As string
body := resp.String()

// As bytes
raw := resp.Body()
```

The body is read once and cached — you can call multiple methods on the same response.

### Status Assertions

All assertions return `*Response` for chaining and call `t.Fatal()` on failure:

```go
resp.AssertStatus(t, 201)      // Any status code
resp.AssertOK(t)               // 200
resp.AssertCreated(t)          // 201
resp.AssertNoContent(t)        // 204
resp.AssertBadRequest(t)       // 400
resp.AssertUnauthorized(t)     // 401
resp.AssertForbidden(t)        // 403
resp.AssertNotFound(t)         // 404
```

### Body Assertions

```go
// Check substring
resp.AssertBodyContains(t, "error")

// Check JSON field (supports dot notation for nested fields)
resp.AssertJSONField(t, "id", 1)
resp.AssertJSONField(t, "user.name", "John")
resp.AssertJSONField(t, "user.address.city", "New York")

// Deserialize JSON with assertion (fails test on parse error)
var user User
resp.AssertJSON(t, &user)
```

### Chaining

```go
ctx := context.Background()

var user User
resp, err := client.Post(ctx, "/users", CreateUserRequest{Name: "John"})
require.NoError(t, err)

resp.
    AssertCreated(t).
    AssertJSONField(t, "name", "John").
    AssertJSON(t, &user)
```

## Full Example

Integration test with PostgreSQL and HTTP API:

```go
func TestCreateUser(t *testing.T) {
    s := suite.New(t)
    ctx := context.Background()

    // Start database
    pg, _ := postgres.New(ctx)
    s.Add(pg)

    // Start your service
    svc := startService(pg.ConnectionString())
    defer svc.Close()

    // Create HTTP client
    client := httpclient.New(httpclient.WithBaseURL(svc.URL))

    // Setup test data
    testground.Apply(t,
        pg.Exec(`CREATE TABLE users (id BIGSERIAL, name TEXT, email TEXT)`),
    )

    // Test create user
    resp, err := client.Post(ctx, "/users", map[string]string{
        "name":  "John",
        "email": "john@example.com",
    })
    require.NoError(t, err)

    var created User
    resp.
        AssertCreated(t).
        AssertJSONField(t, "name", "John").
        AssertJSON(t, &created)

    assert.Equal(t, "John", created.Name)
    assert.Equal(t, "john@example.com", created.Email)
}

func TestGetUser_NotFound(t *testing.T) {
    // ... setup ...
    ctx := context.Background()

    resp, err := client.Get(ctx, "/users/999")
    require.NoError(t, err)

    resp.AssertNotFound(t)
}
```

## Benefits

- **No boilerplate** — JSON serialization, headers, query params handled automatically
- **Fluent assertions** — readable test code with chaining
- **Fail-fast** — assertions call `t.Fatal()` with clear error messages
- **Cached body** — read response multiple times without issues
- **Testcontainers integration** — works seamlessly with testground services