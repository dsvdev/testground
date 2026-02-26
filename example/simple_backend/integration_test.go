package simple_backend_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/dsvdev/testground"
	httpclient "github.com/dsvdev/testground/client/http"
	"github.com/dsvdev/testground/service"
	"github.com/dsvdev/testground/services/postgres"
	"github.com/dsvdev/testground/suite"
)

var (
	pgContainer  *postgres.Container
	svcContainer *service.Container
	client       *httpclient.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	s := suite.NewMain(m)

	net, err := testground.NewNetwork(ctx)
	if err != nil {
		fmt.Printf("failed to create network: %v\n", err)
		os.Exit(1)
	}
	s.Add(net)

	pgContainer, err = postgres.New(ctx,
		postgres.WithNetwork(net),
		postgres.WithNetworkAlias("postgres"),
	)
	if err != nil {
		fmt.Printf("failed to start postgres: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}
	s.Add(pgContainer)

	if err = runMigrations(ctx); err != nil {
		fmt.Printf("failed to run migrations: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}

	svcContainer, err = service.New(ctx,
		service.WithBuildContext("../../"),
		service.WithDockerfile("example/simple_backend/Dockerfile"),
		service.WithNetwork(net),
		service.WithEnv("DATABASE_URL", pgContainer.NetworkConnectionString()),
		service.WithPort("8080"),
	)
	if err != nil {
		fmt.Printf("failed to start service: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}
	s.Add(svcContainer)

	client = httpclient.New(httpclient.WithBaseURL(svcContainer.URL()))

	os.Exit(s.Run())
}

func runMigrations(ctx context.Context) error {
	conn, err := pgContainer.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
		id   BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	return err
}

func TestCreateUser(t *testing.T) {
	s := suite.New(t)
	ctx := context.Background()

	s.BeforeEach(func(ctx context.Context) {
		testground.Apply(t, pgContainer.Exec(`TRUNCATE users RESTART IDENTITY`))
	})

	s.Run("returns 201 with created user", func(t *testing.T) {
		resp, err := client.Post(ctx, "/users", map[string]string{"name": "Alice"})
		if err != nil {
			t.Fatal(err)
		}

		var user struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		resp.AssertCreated(t).AssertJSON(t, &user)

		if user.Name != "Alice" {
			t.Errorf("expected Name=Alice, got %q", user.Name)
		}
		if user.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})
}

func TestGetUser(t *testing.T) {
	s := suite.New(t)
	ctx := context.Background()

	s.BeforeEach(func(ctx context.Context) {
		testground.Apply(t, pgContainer.Exec(`TRUNCATE users RESTART IDENTITY`))
	})

	s.Run("returns 200 with existing user", func(t *testing.T) {
		testground.Apply(t,
			pgContainer.Exec(`INSERT INTO users (name) VALUES (@name)`, pgx.NamedArgs{"name": "Bob"}),
		)

		resp, err := client.Get(ctx, "/users/1")
		if err != nil {
			t.Fatal(err)
		}

		var user struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		resp.AssertOK(t).AssertJSON(t, &user)

		if user.Name != "Bob" {
			t.Errorf("expected Name=Bob, got %q", user.Name)
		}
	})
}

func TestGetUser_NotFound(t *testing.T) {
	s := suite.New(t)
	ctx := context.Background()

	s.BeforeEach(func(ctx context.Context) {
		testground.Apply(t, pgContainer.Exec(`TRUNCATE users RESTART IDENTITY`))
	})

	s.Run("returns 404 for missing user", func(t *testing.T) {
		resp, err := client.Get(ctx, "/users/9999")
		if err != nil {
			t.Fatal(err)
		}

		resp.AssertNotFound(t)
	})
}
