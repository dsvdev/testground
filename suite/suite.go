package suite

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type Managed interface {
	Terminate(ctx context.Context) error
}

type MainSuite struct {
	m          *testing.M
	mu         sync.Mutex
	containers []Managed
}

func NewMain(m *testing.M) *MainSuite {
	return &MainSuite{m: m}
}

func (s *MainSuite) Add(c Managed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.containers = append(s.containers, c)
}

func (s *MainSuite) Cleanup() {
	s.mu.Lock()
	containers := s.containers
	s.mu.Unlock()

	ctx := context.Background()
	for i := len(containers) - 1; i >= 0; i-- {
		if err := containers[i].Terminate(ctx); err != nil {
			fmt.Printf("warning: failed to terminate container: %v\n", err)
		}
	}
}

func (s *MainSuite) Run() int {
	code := s.m.Run()
	s.Cleanup()
	return code
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
	started    bool
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("suite: BeforeAll must be called before the first Run")
	}
	s.beforeAll = append(s.beforeAll, fn)
}

func (s *Suite) AfterAll(fn func(ctx context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("suite: AfterAll must be called before the first Run")
	}
	s.afterAll = append(s.afterAll, fn)
}

func (s *Suite) BeforeEach(fn func(ctx context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("suite: BeforeEach must be called before the first Run")
	}
	s.beforeEach = append(s.beforeEach, fn)
}

func (s *Suite) AfterEach(fn func(ctx context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("suite: AfterEach must be called before the first Run")
	}
	s.afterEach = append(s.afterEach, fn)
}

func (s *Suite) Run(name string, fn func(t *testing.T)) {
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
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
