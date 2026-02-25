package testground

import (
	"context"
	"testing"
)

type Precondition interface {
	Apply(ctx context.Context, t *testing.T) error
}

func Apply(t *testing.T, preconditions ...Precondition) {
	t.Helper()
	ctx := context.Background()
	for _, p := range preconditions {
		if err := p.Apply(ctx, t); err != nil {
			t.Fatalf("precondition failed: %v", err)
		}
	}
}
