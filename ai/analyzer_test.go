package ai_test

import (
	"slices"
	"testing"

	"github.com/dsvdev/testground/ai"
)

func TestAnalyze_SimpleBackend(t *testing.T) {
	ctx, err := ai.Analyze("../example/simple_backend")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Endpoints
	if len(ctx.Endpoints) < 2 {
		t.Errorf("expected at least 2 endpoints, got %d", len(ctx.Endpoints))
	}

	hasGET := slices.ContainsFunc(ctx.Endpoints, func(e ai.EndpointInfo) bool {
		return e.Method == "GET" && e.Path == "/users/{userId}"
	})
	if !hasGET {
		t.Errorf("expected GET /users/{userId} endpoint, got %+v", ctx.Endpoints)
	}

	hasPOST := slices.ContainsFunc(ctx.Endpoints, func(e ai.EndpointInfo) bool {
		return e.Method == "POST" && e.Path == "/users"
	})
	if !hasPOST {
		t.Errorf("expected POST /users endpoint, got %+v", ctx.Endpoints)
	}

	// Models
	hasUser := slices.ContainsFunc(ctx.Models, func(m ai.ModelInfo) bool {
		return m.Name == "User"
	})
	if !hasUser {
		t.Errorf("expected User model, got %+v", ctx.Models)
	}

	// Tables — extracted from Query calls in user_repo.go
	if !slices.Contains(ctx.Tables, "users") {
		t.Errorf("expected table 'users', got %v", ctx.Tables)
	}

	// Topics — simple_backend has no Kafka
	if len(ctx.Topics) != 0 {
		t.Errorf("expected no Kafka topics, got %v", ctx.Topics)
	}
}
