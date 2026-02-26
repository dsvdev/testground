# faker

The `faker` package provides cryptographically secure random data generators for use in integration tests.
All functions use `crypto/rand` — no seed required, safe for concurrent use.

## Installation

```go
import "github.com/dsvdev/testground/faker"
```

## Functions

### `RandomInt(min, max int) int`

Returns a random integer in the closed range `[min, max]`. Panics if `min > max`.

```go
age  := faker.RandomInt(18, 65)
port := faker.RandomInt(1024, 65535)
```

### `RandomInt64(min, max int64) int64`

Returns a random `int64` in the closed range `[min, max]`. Panics if `min > max`.

```go
timestamp := faker.RandomInt64(0, 1_000_000_000)
```

### `RandomString(length int) string`

Returns a random lowercase ASCII string (`a-z`) of the given length.
Returns an empty string when `length` is `0`.

```go
name  := faker.RandomString(10)
token := faker.RandomString(32)
```

### `RandomUUID() string`

Returns a randomly generated UUID v4 string in the canonical format
`xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx` with version (`4`) and variant (`10xx`) bits
set according to RFC 4122.

```go
id := faker.RandomUUID()
```

## Example

```go
func TestCreateUser(t *testing.T) {
    name := faker.RandomString(10)
    age  := faker.RandomInt(18, 65)
    id   := faker.RandomUUID()

    testground.Apply(t,
        pg.Exec(`INSERT INTO users (id, name, age) VALUES (@id, @name, @age)`,
            pgx.NamedArgs{"id": id, "name": name, "age": age},
        ),
    )

    resp, err := client.Get(ctx, "/users/"+id)
    // ...
}
```

## Running tests

```sh
go test ./faker/...
```