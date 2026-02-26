package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/dsvdev/testground/client/http"
)

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users" {
			t.Errorf("expected /users, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "name": "John"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/users")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var data map[string]any
		json.Unmarshal(body, &data)

		if data["name"] != "John" {
			t.Errorf("expected name=John, got %v", data["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1, "name": "John"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Post(ctx, "/users", map[string]string{"name": "John"})
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	resp.AssertCreated(t)
}

func TestClient_WithQueryParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		limit := r.URL.Query().Get("limit")

		if page != "1" {
			t.Errorf("expected page=1, got %s", page)
		}
		if limit != "10" {
			t.Errorf("expected limit=10, got %s", limit)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/users",
		client.WithQueryParam("page", "1"),
		client.WithQueryParam("limit", "10"),
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestClient_WithBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret-token" {
			t.Errorf("expected Authorization: Bearer my-secret-token, got %s", auth)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(
		client.WithBaseURL(server.URL),
		client.WithBearerToken("my-secret-token"),
	)
	resp, err := c.Get(ctx, "/protected")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestClient_WithRequestHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custom := r.Header.Get("X-Custom-Header")
		if custom != "custom-value" {
			t.Errorf("expected X-Custom-Header: custom-value, got %s", custom)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/test",
		client.WithRequestHeader("X-Custom-Header", "custom-value"),
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestResponse_AssertStatus_Fails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// AssertNotFound should pass
	resp.AssertNotFound(t)
}

func TestResponse_AssertJSONField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"name": "John",
			"active": true,
			"address": {
				"city": "New York",
				"zip": "10001"
			}
		}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/user")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.
		AssertOK(t).
		AssertJSONField(t, "id", 1).
		AssertJSONField(t, "name", "John").
		AssertJSONField(t, "active", true).
		AssertJSONField(t, "address.city", "New York").
		AssertJSONField(t, "address.zip", "10001")
}

func TestResponse_AssertBodyContains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello, World!"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/hello")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.
		AssertOK(t).
		AssertBodyContains(t, "Hello").
		AssertBodyContains(t, "World")
}

func TestResponse_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 42, "name": "Test User"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/user")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var user User
	resp.AssertOK(t)

	if err := resp.JSON(&user); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	if user.ID != 42 {
		t.Errorf("expected ID=42, got %d", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("expected Name='Test User', got %s", user.Name)
	}
}

func TestResponse_AssertJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 42, "name": "Test User"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Get(ctx, "/user")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var user User
	resp.
		AssertOK(t).
		AssertJSON(t, &user)

	if user.ID != 42 {
		t.Errorf("expected ID=42, got %d", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("expected Name='Test User', got %s", user.Name)
	}
}

func TestClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Put(ctx, "/users/1", map[string]string{"name": "Updated"})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestClient_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Patch(ctx, "/users/1", map[string]string{"name": "Patched"})
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}

	resp.AssertOK(t)
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Delete(ctx, "/users/1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	resp.AssertNoContent(t)
}

func TestResponse_Chaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1, "name": "John"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	c := client.New(client.WithBaseURL(server.URL))
	resp, err := c.Post(ctx, "/users", map[string]string{"name": "John"})
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var user User
	resp.
		AssertCreated(t).
		AssertJSONField(t, "name", "John").
		AssertJSON(t, &user)

	if user.ID != 1 {
		t.Errorf("expected ID=1, got %d", user.ID)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c := client.New(client.WithBaseURL(server.URL))
	_, err := c.Get(ctx, "/test")
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}
