# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-02-27

Initial release of **testground** — a Go integration testing framework for spinning up real
infrastructure (databases, message brokers, custom services) directly from test code using
Docker containers. No mocks, no Docker Compose files, no shared test environments.

### Added

#### Test Suite Management (`suite` package)

- **`Suite`** — manages container lifecycle and hooks for individual tests (`Test*` functions):
  - `New(t *testing.T) *Suite` — creates a suite bound to a test
  - `Add(c Managed)` — registers containers for automatic cleanup in reverse order
  - `Run(name, fn)` — runs subtests with hook support
  - Lifecycle hooks: `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`
  - Panic guard against late hook registration

- **`MainSuite`** — manages container lifecycle across the entire test binary (`TestMain`):
  - `NewMain(m *testing.M)` — creates suite for `TestMain`
  - `Run() int` — runs all tests and terminates containers afterward
  - `Cleanup()` — early-exit cleanup without running tests
  - Safe error handling for early-exit scenarios

#### Preconditions System

- **`Precondition`** interface — declarative test data setup:
  - `interface { Apply(ctx context.Context, t *testing.T) error }`
  - `Apply(t, preconditions...)` — applies multiple preconditions with fail-fast semantics

#### Docker Network (`network.go`)

- **`Network`** — shared Docker bridge network for container-to-container communication:
  - `NewNetwork(ctx)` — creates a bridge network labeled `testground`
  - `Name()` — returns the network name for container attachment
  - `Terminate(ctx)` — removes the network
  - Enables DNS-based service discovery between containers via aliases

#### PostgreSQL Container (`services/postgres`)

- `New(ctx, opts...)` — creates and starts a PostgreSQL container
- `Terminate(ctx)` — stops and removes the container
- Configuration options:
  - `WithVersion(v)` — PostgreSQL image version (default: `"16"`)
  - `WithDatabase(d)` — database name (default: `"test"`)
  - `WithUser(u)` — user name (default: `"test"`)
  - `WithPassword(p)` — password (default: `"test"`)
  - `WithPort(p)` — fixed host port; random free port by default
  - `WithNetwork(n)` — attach to a Docker network
  - `WithNetworkAlias(alias)` — set DNS alias within the network
- Connection helpers:
  - `ConnectionString()` — external DSN: `postgres://user:password@host:port/database?sslmode=disable`
  - `NetworkConnectionString()` — internal DSN for containers on the same network
  - `Conn(ctx)` — returns a new `pgx.Conn` (caller is responsible for closing)
  - `Pool(ctx)` — returns a lazy-initialized `pgxpool.Pool` (closed automatically on `Terminate`)
- Precondition:
  - `Exec(sql, args...)` — executes SQL with optional named arguments (`pgx.NamedArgs`); supports schema creation, seed data, and migrations

#### Kafka Container (`services/kafka`)

- `New(ctx, opts...)` — creates a Zookeeper + Kafka container pair with an isolated internal network
- `Terminate(ctx)` — stops Kafka, then Zookeeper, then the internal network
- Configuration options:
  - `WithVersion(v)` — Confluent Platform image version (default: `"7.6.1"`)
  - `WithNetwork(n)` — attach Kafka to an external Docker network
  - `WithNetworkAlias(alias)` — Kafka alias within the network (default: `"kafka"`)
- Bootstrap server helpers:
  - `BootstrapServers()` — external address for host test code (`host:port`)
  - `NetworkBootstrapServers()` — internal address for containers (`kafka:9092`)
- Preconditions:
  - `CreateTopic(topic, opts...)` — creates a Kafka topic (no-op if already exists):
    - `WithPartitions(n)` — number of partitions (default: `1`)
    - `WithReplicationFactor(n)` — replication factor (default: `1`)
  - `Publish(topic, value)` — publishes a single message to a topic; auto-creates topic if configured
- Assertions (read from the beginning, 30 s timeout):
  - `AssertMessageCount(t, topic, count)` — asserts exact number of messages
  - `AssertHasMessage(t, topic, value)` — asserts a message with exact byte value exists
  - `AssertHasMessageContaining(t, topic, substr, wantCount)` — asserts substring match with expected occurrence count

#### Custom Service Container (`service` package)

- `New(ctx, opts...)` — builds and starts a Docker image from a local Dockerfile
- `URL()` — returns service HTTP URL: `http://host:port`
- `Port()` — returns the mapped port number
- `Terminate(ctx)` — stops and removes the container
- Configuration options:
  - `WithDockerfile(path)` — path to the `Dockerfile`
  - `WithBuildContext(path)` — build context directory
  - `WithEnv(key, value)` — set an environment variable
  - `WithPort(port)` — expose a port (default: `8080`)
  - `WithNetwork(n)` — attach to a Docker network
  - `WithWaitFor(strategy)` — custom wait strategy (default: wait for port `8080`)

#### HTTP Client (`client/httpclient`)

Fluent HTTP client with built-in assertions for testing APIs.

- `New(opts...)` — creates an HTTP client
- Request methods: `Get`, `Post`, `Put`, `Patch`, `Delete`
- Client configuration:
  - `WithBaseURL(url)` — base URL for all requests
  - `WithTimeout(d)` — request timeout (default: `30s`)
  - `WithHeader(key, value)` — global default header
  - `WithBearerToken(token)` — sets the `Authorization: Bearer <token>` header
- Per-request options:
  - `WithQueryParam(key, value)` — appends a query parameter
  - `WithRequestHeader(key, value)` — adds a request-scoped header
- Response body access (body is cached for multiple reads):
  - `JSON(target)` — deserializes body into a struct
  - `String()` — returns body as a string
  - `Body()` — returns raw bytes
- Status assertions (chainable):
  - `AssertStatus(t, code)` — any HTTP status code
  - `AssertOK(t)` — `200 OK`
  - `AssertCreated(t)` — `201 Created`
  - `AssertNoContent(t)` — `204 No Content`
  - `AssertBadRequest(t)` — `400 Bad Request`
  - `AssertUnauthorized(t)` — `401 Unauthorized`
  - `AssertForbidden(t)` — `403 Forbidden`
  - `AssertNotFound(t)` — `404 Not Found`
- Body assertions (chainable):
  - `AssertBodyContains(t, substr)` — substring match
  - `AssertJSONField(t, path, expected)` — dot-notation JSON path with type-aware comparison (e.g. `"user.address.city"`)
  - `AssertJSON(t, target)` — deserialize and assert in one call

#### Test Data Generation (`faker` package)

Cryptographically-secure random test data generators backed by `crypto/rand`. All functions are concurrent-safe.

- `RandomInt(min, max)` — random `int` in the range `[min, max]`
- `RandomInt64(min, max)` — random `int64` in the range `[min, max]`
- `RandomString(length)` — random lowercase ASCII string (`a–z`)
- `RandomUUID()` — RFC 4122 UUID v4 with correct version and variant bits