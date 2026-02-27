package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dsvdev/testground"
	"github.com/dsvdev/testground/faker"
	"github.com/dsvdev/testground/services/postgres"
)

// Example: user-defined precondition factory
// In real projects this would live in testdata/preconditions.go
func insertUser(pg *postgres.Container) func(name string) testground.Precondition {
	return func(name string) testground.Precondition {
		return pg.Exec(
			`INSERT INTO users (name) VALUES (@name)`,
			pgx.NamedArgs{"name": name},
		)
	}
}

func TestExecPrecondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pg, err := postgres.New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		pg.Terminate(context.Background())
	})

	// Create precondition factory bound to this container
	InsertUser := insertUser(pg)

	// Apply preconditions: create table and insert data
	testground.Apply(t,
		pg.Exec(`
			CREATE TABLE users (
				id   BIGSERIAL PRIMARY KEY,
				name TEXT NOT NULL
			)
		`),
		InsertUser("Alice"),
		InsertUser("Bob"),
	)

	// Verify data was inserted
	conn, err := pg.Conn(ctx)
	if err != nil {
		t.Fatalf("Conn() error = %v", err)
	}
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("SELECT COUNT(*) error = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	var name string
	err = conn.QueryRow(ctx, "SELECT name FROM users WHERE name = 'Alice'").Scan(&name)
	if err != nil {
		t.Fatalf("SELECT name error = %v", err)
	}
	if name != "Alice" {
		t.Errorf("name = %q, want %q", name, "Alice")
	}

	// Verify non-existent user returns no rows
	err = conn.QueryRow(ctx, "SELECT name FROM users WHERE name = 'Charlie'").Scan(&name)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("SELECT non-existent user: got err = %v, want pgx.ErrNoRows", err)
	}
}

func TestExecPrecondition_PoolReuse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pg, err := postgres.New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { pg.Terminate(context.Background()) })

	insertUser := insertUser(pg)

	preconditions := []testground.Precondition{
		pg.Exec(`CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL)`),
	}
	for range 10 {
		preconditions = append(preconditions, insertUser(faker.RandomString(8)))
	}

	testground.Apply(t, preconditions...)

	pool, err := pg.Pool(ctx)
	if err != nil {
		t.Fatalf("Pool() error = %v", err)
	}

	stat := pool.Stat()
	if stat.TotalConns() > 2 {
		t.Errorf("TotalConns = %d after 10 sequential preconditions, want <= 2", stat.TotalConns())
	}
}
