package suite

import (
	"context"
	"sync"
	"testing"
)

type Managed interface {
	Terminate(ctx context.Context) error
}

type Suite struct {
	t          *testing.T
	mu         sync.Mutex
	containers []Managed

	beforeAll  []func(ctx context.Context)
	afterAll   []func(ctx context.Context)
	beforeEach []func(ctx context.Context)
	afterEach  []func(ctx context.Context)

	beforeOnce sync.Once
}

func New(t *testing.T) *Suite {
	s := &Suite{t: t}

	t.Cleanup(func() {
		ctx := context.Background()

		// Call AfterAll hooks
		for _, hook := range s.afterAll {
			hook(ctx)
		}

		// Terminate all containers in reverse order
		s.mu.Lock()
		containers := s.containers
		s.mu.Unlock()

		for i := len(containers) - 1; i >= 0; i-- {
			if err := containers[i].Terminate(ctx); err != nil {
				s.t.Logf("warning: failed to terminate container: %v", err)
			}
		}
	})

	return s
}

func (s *Suite) Add(c Managed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.containers = append(s.containers, c)
}

func (s *Suite) BeforeAll(fn func(ctx context.Context)) {
	s.beforeAll = append(s.beforeAll, fn)
}

func (s *Suite) AfterAll(fn func(ctx context.Context)) {
	s.afterAll = append(s.afterAll, fn)
}

func (s *Suite) BeforeEach(fn func(ctx context.Context)) {
	s.beforeEach = append(s.beforeEach, fn)
}

func (s *Suite) AfterEach(fn func(ctx context.Context)) {
	s.afterEach = append(s.afterEach, fn)
}

func (s *Suite) Run(name string, fn func(t *testing.T)) {
	s.t.Run(name, func(t *testing.T) {
		ctx := context.Background()

		// Call BeforeAll once before the first test
		s.beforeOnce.Do(func() {
			for _, hook := range s.beforeAll {
				hook(ctx)
			}
		})

		// Call BeforeEach hooks
		for _, hook := range s.beforeEach {
			hook(ctx)
		}

		// Register AfterEach to run after the test
		t.Cleanup(func() {
			for _, hook := range s.afterEach {
				hook(ctx)
			}
		})

		fn(t)
	})
}
