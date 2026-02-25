package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"testground/services/postgres"
)

func TestPostgresContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := postgres.New(ctx,
		postgres.WithVersion("16"),
		postgres.WithDatabase("testdb"),
		postgres.WithUser("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithPort("5432"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("Terminate() error = %v", err)
		}
	})

	connStr := container.ConnectionString()
	if connStr == "" {
		t.Fatal("ConnectionString() returned empty string")
	}

	t.Logf("ConnectionString: %s", connStr)

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("pgx.Connect() error = %v", err)
	}
	defer conn.Close(ctx)

	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("SELECT 1 error = %v", err)
	}

	if result != 1 {
		t.Errorf("SELECT 1 = %d, want 1", result)
	}
}

func TestPostgresContainer_RandomPorts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Start two containers without specifying port
	container1, err := postgres.New(ctx, postgres.WithDatabase("db1"))
	if err != nil {
		t.Fatalf("New() container1 error = %v", err)
	}
	t.Cleanup(func() {
		container1.Terminate(context.Background())
	})

	container2, err := postgres.New(ctx, postgres.WithDatabase("db2"))
	if err != nil {
		t.Fatalf("New() container2 error = %v", err)
	}
	t.Cleanup(func() {
		container2.Terminate(context.Background())
	})

	connStr1 := container1.ConnectionString()
	connStr2 := container2.ConnectionString()

	t.Logf("Container1: %s", connStr1)
	t.Logf("Container2: %s", connStr2)

	// Ports must be different
	if connStr1 == connStr2 {
		t.Fatal("containers got the same connection string, expected different ports")
	}

	// Both should be connectable
	conn1, err := pgx.Connect(ctx, connStr1)
	if err != nil {
		t.Fatalf("pgx.Connect() container1 error = %v", err)
	}
	defer conn1.Close(ctx)

	conn2, err := pgx.Connect(ctx, connStr2)
	if err != nil {
		t.Fatalf("pgx.Connect() container2 error = %v", err)
	}
	defer conn2.Close(ctx)

	// Verify both work
	var r1, r2 int
	if err := conn1.QueryRow(ctx, "SELECT 1").Scan(&r1); err != nil {
		t.Fatalf("SELECT 1 on container1 error = %v", err)
	}
	if err := conn2.QueryRow(ctx, "SELECT 1").Scan(&r2); err != nil {
		t.Fatalf("SELECT 1 on container2 error = %v", err)
	}

	if r1 != 1 || r2 != 1 {
		t.Errorf("got r1=%d, r2=%d, want both 1", r1, r2)
	}
}
