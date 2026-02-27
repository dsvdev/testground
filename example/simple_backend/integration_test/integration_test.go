package integration_test

import (
	"context"
	"fmt"
	"github.com/dsvdev/testground/faker"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/dsvdev/testground"
	"github.com/dsvdev/testground/suite"
)

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

func TestCreateAndGetUser(t *testing.T) {
	s := suite.New(t)
	ctx := context.Background()

	s.BeforeEach(func(ctx context.Context) {
		testground.Apply(t, pgContainer.Exec(`TRUNCATE users RESTART IDENTITY`))
	})

	s.Run("full user way", func(t *testing.T) {
		userName := faker.RandomString(faker.RandomInt(5, 10))
		t.Logf("creating user: %s", userName)
		resp, err := client.Post(ctx, "/users", map[string]string{"name": userName})
		if err != nil {
			t.Fatal(err)
		}
		var user struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		resp.AssertCreated(t).AssertJSON(t, &user)
		t.Logf("user created succesfully: %v", user)
		userID := user.ID

		if user.Name != userName {
			t.Errorf("expected Name=%s, got %q", userName, user.Name)
		}

		t.Logf("getting user by id: %d", userID)
		resp, err = client.Get(ctx, fmt.Sprintf("/users/%d", userID))
		if err != nil {
			t.Fatal(err)
		}

		resp.AssertOK(t).AssertJSON(t, &user)
		t.Logf("getting user by id successfully: %v", user)

		if user.Name != userName {
			t.Errorf("expected Name=%s, got %q", userName, user.Name)
		}
		if user.ID != userID {
			t.Errorf("expected ID=%d, got %d", userID, user.ID)
		}
	})
}
