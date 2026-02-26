package suite_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"testground/services/postgres"
	"testground/suite"
)

// mockManaged tracks whether Terminate was called
type mockManaged struct {
	terminated bool
	mu         sync.Mutex
}

func (m *mockManaged) Terminate(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminated = true
	return nil
}

func (m *mockManaged) isTerminated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.terminated
}

func TestSuite_ContainerTerminatedAfterSuite(t *testing.T) {
	mock := &mockManaged{}

	// Run in a subtest so we can check after it completes
	t.Run("inner", func(t *testing.T) {
		s := suite.New(t)
		s.Add(mock)

		// Container should not be terminated yet
		if mock.isTerminated() {
			t.Error("container terminated too early")
		}
	})

	// After the subtest completes, container should be terminated
	if !mock.isTerminated() {
		t.Error("container was not terminated after suite")
	}
}

func TestSuite_HooksCalledInOrder(t *testing.T) {
	var order []string
	var mu sync.Mutex

	record := func(event string) {
		mu.Lock()
		order = append(order, event)
		mu.Unlock()
	}

	t.Run("suite", func(t *testing.T) {
		s := suite.New(t)

		s.BeforeAll(func(ctx context.Context) {
			record("BeforeAll")
		})

		s.AfterAll(func(ctx context.Context) {
			record("AfterAll")
		})

		s.BeforeEach(func(ctx context.Context) {
			record("BeforeEach")
		})

		s.AfterEach(func(ctx context.Context) {
			record("AfterEach")
		})

		s.Run("test1", func(t *testing.T) {
			record("test1")
		})

		s.Run("test2", func(t *testing.T) {
			record("test2")
		})
	})

	expected := []string{
		"BeforeAll",
		"BeforeEach", "test1", "AfterEach",
		"BeforeEach", "test2", "AfterEach",
		"AfterAll",
	}

	mu.Lock()
	defer mu.Unlock()

	if len(order) != len(expected) {
		t.Fatalf("got %d events, want %d: %v", len(order), len(expected), order)
	}

	for i, event := range expected {
		if order[i] != event {
			t.Errorf("event[%d] = %q, want %q", i, order[i], event)
		}
	}
}

func TestSuite_IsolatedContainers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var port1, port2 string
	var mu sync.Mutex

	t.Run("outer", func(t *testing.T) {
		s := suite.New(t)

		s.Run("test A", func(t *testing.T) {
			inner := suite.New(t)
			pg, err := postgres.New(ctx)
			if err != nil {
				t.Fatalf("postgres.New() error = %v", err)
			}
			inner.Add(pg)

			mu.Lock()
			port1 = pg.ConnectionString()
			mu.Unlock()
		})

		// After test A, check if first container is gone
		mu.Lock()
		// We can't directly check termination, but we can verify
		// the second container gets a different port
		mu.Unlock()

		s.Run("test B", func(t *testing.T) {
			inner := suite.New(t)
			pg, err := postgres.New(ctx)
			if err != nil {
				t.Fatalf("postgres.New() error = %v", err)
			}
			inner.Add(pg)

			mu.Lock()
			port2 = pg.ConnectionString()
			mu.Unlock()
		})
	})

	mu.Lock()
	defer mu.Unlock()

	// Containers should have different connection strings (different ports)
	if port1 == port2 {
		t.Errorf("containers have same connection string, expected different ports")
	}

	t.Logf("Container A: %s", port1)
	t.Logf("Container B: %s", port2)
}
