package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dsvdev/testground/ai"
)

const validStoriesJSON = `[
  {
    "title": "Create user happy path",
    "description": "POST /users creates a new user and returns 201",
    "steps": ["Send POST /users with name", "Assert 201 response", "Assert user in DB"]
  },
  {
    "title": "Get user by ID",
    "description": "GET /users/{id} returns existing user",
    "steps": ["Create user", "Send GET /users/1", "Assert 200 with user data"]
  },
  {
    "title": "Get user not found",
    "description": "GET /users/{id} returns 404 for missing user",
    "steps": ["Send GET /users/999", "Assert 404 response"]
  }
]`

type mockLLM struct {
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	content   string
	toolCalls []ai.ToolCall
	done      bool
	err       error
}

func newMockLLM(responses ...mockResponse) *mockLLM {
	return &mockLLM{responses: responses}
}

func textResponse(content string) mockResponse {
	return mockResponse{content: content, done: true}
}

func (m *mockLLM) Complete(_ context.Context, _ []ai.Message, _ []ai.Tool) (*ai.Response, error) {
	i := m.callCount
	m.callCount++
	if i >= len(m.responses) {
		return &ai.Response{Content: "", Done: true}, nil
	}
	r := m.responses[i]
	if r.err != nil {
		return nil, r.err
	}
	return &ai.Response{
		Content:   r.content,
		ToolCalls: r.toolCalls,
		Done:      r.done,
	}, nil
}

func TestGenerate_ValidJSON(t *testing.T) {
	mock := newMockLLM(textResponse(validStoriesJSON))
	project := &ai.ProjectContext{
		Endpoints: []ai.EndpointInfo{
			{Method: "POST", Path: "/users", Handler: "<anonymous>"},
			{Method: "GET", Path: "/users/{userId}", Handler: "<anonymous>"},
		},
		Tables: []string{"users"},
	}

	stories, err := ai.Generate(context.Background(), mock, project)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(stories) != 3 {
		t.Errorf("expected 3 stories, got %d", len(stories))
	}
	for i, s := range stories {
		if s.Title == "" {
			t.Errorf("story[%d] has empty Title", i)
		}
		if s.Description == "" {
			t.Errorf("story[%d] has empty Description", i)
		}
		if len(s.Steps) == 0 {
			t.Errorf("story[%d] has no Steps", i)
		}
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 LLM call, got %d", mock.callCount)
	}
}

func TestGenerate_RetryOnInvalidJSON(t *testing.T) {
	mock := newMockLLM(textResponse("not json at all"), textResponse(validStoriesJSON))
	project := &ai.ProjectContext{}

	stories, err := ai.Generate(context.Background(), mock, project)
	if err != nil {
		t.Fatalf("Generate failed on retry: %v", err)
	}
	if len(stories) == 0 {
		t.Error("expected non-empty stories after retry")
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 LLM calls (retry), got %d", mock.callCount)
	}
}

func TestGenerate_ErrorOnDoubleInvalidJSON(t *testing.T) {
	mock := newMockLLM(textResponse("not json"), textResponse("also not json"))
	project := &ai.ProjectContext{}

	_, err := ai.Generate(context.Background(), mock, project)
	if err == nil {
		t.Error("expected error when both responses are invalid JSON")
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mock.callCount)
	}
}

func TestGenerate_LLMError(t *testing.T) {
	mock := newMockLLM(mockResponse{err: errors.New("network error")})
	project := &ai.ProjectContext{}

	_, err := ai.Generate(context.Background(), mock, project)
	if err == nil {
		t.Error("expected error on LLM failure")
	}
}
