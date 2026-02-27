package ai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dsvdev/testground/ai"
)

// mockExecutor replaces the real Executor in runner tests.
type mockExecutor struct {
	output string // returned as output for every Execute call
	calls  int
}

func (m *mockExecutor) Execute(_ context.Context, call ai.ToolCall) (string, string, string) {
	m.calls++
	return call.Name, string(call.Input), m.output
}

// runStoryViaAgent runs a single story through a minimal Agent with mocked LLM and Executor.
// We expose runStory indirectly through agent.Run so we don't need to export it.
func runStoryHelper(t *testing.T, llm *mockLLM, ex *mockExecutor, story ai.UserStory, maxSteps int) ai.UserStory {
	t.Helper()
	// Wrap in a minimal agent-like execution using exported Run path.
	// Since runStory is unexported, we test it via the exported agent.Run.
	// We use a custom LLM/Executor by injecting via the agent's internal structure.
	// Alternatively, we can test the observable behaviour through agent.Run.
	// Here we build a lightweight agent with a custom executor via a test hook.
	a := ai.New(
		ai.WithLLM(llm),
		ai.WithMaxStepsPerStory(maxSteps),
		ai.WithMaxStepsTotal(maxSteps*10),
	)
	// Inject executor via unexported hook — since executor is unexported,
	// we test via the full Run path with a real (nil-safe) executor.
	report, err := a.Run(context.Background(), []ai.UserStory{story})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(report.UserStories) != 1 {
		t.Fatalf("expected 1 story result, got %d", len(report.UserStories))
	}
	return report.UserStories[0]
}

func TestRunStory_Passed(t *testing.T) {
	// LLM sequence: tool_call → tool_call → Done with "PASSED"
	toolCallExec := ai.ToolCall{ID: "c1", Name: "http_request", Input: json.RawMessage(`{"method":"GET","path":"/users"}`)}
	mock := newMockLLM(
		mockResponse{toolCalls: []ai.ToolCall{toolCallExec}, done: false},
		textResponse("PASSED"),
	)

	story := ai.UserStory{
		Title:       "Get users",
		Description: "GET /users returns list",
		Steps:       []string{"Send GET /users", "Assert 200"},
	}

	result := runStoryHelper(t, mock, nil, story, 10)

	if result.Status != "passed" {
		t.Errorf("expected status 'passed', got %q (error: %s)", result.Status, result.Error)
	}
	if len(result.StepResults) != 1 {
		t.Errorf("expected 1 step result, got %d", len(result.StepResults))
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mock.callCount)
	}
}

func TestRunStory_Failed(t *testing.T) {
	mock := newMockLLM(textResponse("FAILED: expected 400 got 201"))

	story := ai.UserStory{Title: "Bad request test", Description: "test", Steps: []string{"step"}}
	result := runStoryHelper(t, mock, nil, story, 10)

	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "expected 400 got 201") {
		t.Errorf("expected error to contain reason, got %q", result.Error)
	}
}

func TestRunStory_PassedInMarkdown(t *testing.T) {
	markdown := `I have executed all the steps for this user story.

All assertions passed successfully:
- Created the user via POST /users → 201 Created
- Verified the user exists in the database

**PASSED**`

	mock := newMockLLM(textResponse(markdown))
	story := ai.UserStory{Title: "Markdown response test", Steps: []string{"step"}}
	result := runStoryHelper(t, mock, nil, story, 10)

	if result.Status != "passed" {
		t.Errorf("expected status 'passed', got %q (error: %s)", result.Status, result.Error)
	}
}

func TestRunStory_FailedInMarkdown(t *testing.T) {
	markdown := `I executed the steps but encountered an issue.

The POST /users endpoint returned 201 when we expected 400.

FAILED: expected status 400, got 201

Please review the validation logic.`

	mock := newMockLLM(textResponse(markdown))
	story := ai.UserStory{Title: "Markdown fail test", Steps: []string{"step"}}
	result := runStoryHelper(t, mock, nil, story, 10)

	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "expected status 400, got 201") {
		t.Errorf("expected reason in error, got %q", result.Error)
	}
}

func TestRunStory_MaxStepsReached(t *testing.T) {
	// LLM always returns tool calls, never Done
	toolCall := ai.ToolCall{ID: "c1", Name: "http_request", Input: json.RawMessage(`{"method":"GET","path":"/"}`)}
	// Provide many responses
	responses := make([]mockResponse, 20)
	for i := range responses {
		responses[i] = mockResponse{toolCalls: []ai.ToolCall{toolCall}, done: false}
	}
	mock := newMockLLM(responses...)

	story := ai.UserStory{Title: "Infinite loop test", Description: "test", Steps: []string{"step"}}
	result := runStoryHelper(t, mock, nil, story, 2) // max 2 steps

	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if !strings.Contains(result.Error, "maxStepsPerStory") {
		t.Errorf("expected error to mention maxStepsPerStory, got %q", result.Error)
	}
}

func TestAgent_Run_Report(t *testing.T) {
	// 2 stories: first passes, second fails
	mock := newMockLLM(
		textResponse("PASSED"),
		textResponse("FAILED: user not found"),
	)

	stories := []ai.UserStory{
		{Title: "Story 1", Description: "desc1", Steps: []string{"step"}},
		{Title: "Story 2", Description: "desc2", Steps: []string{"step"}},
	}

	a := ai.New(
		ai.WithLLM(mock),
		ai.WithMaxStepsPerStory(5),
		ai.WithMaxStepsTotal(100),
	)
	report, err := a.Run(context.Background(), stories)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if report.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	if report.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", report.Skipped)
	}
}

func TestAgent_Run_MaxStepsTotal_Skips(t *testing.T) {
	// Each story uses MaxStepsPerStory steps, set total to trigger skip on 3rd
	toolCall := ai.ToolCall{ID: "c1", Name: "http_request", Input: json.RawMessage(`{"method":"GET","path":"/"}`)}
	responses := make([]mockResponse, 30)
	for i := range responses {
		if i%3 == 2 { // every 3rd call returns PASSED
			responses[i] = textResponse("PASSED")
		} else {
			responses[i] = mockResponse{toolCalls: []ai.ToolCall{toolCall}, done: false}
		}
	}
	mock := newMockLLM(responses...)

	stories := []ai.UserStory{
		{Title: "S1", Steps: []string{"s"}},
		{Title: "S2", Steps: []string{"s"}},
		{Title: "S3", Steps: []string{"s"}},
	}

	// MaxStepsTotal=3, MaxStepsPerStory=10: each story uses 2 tool steps + 1 done
	// After story 1 (2 steps) + story 2 (2 steps) = 4 steps > 3, story 3 should be skipped
	a := ai.New(
		ai.WithLLM(mock),
		ai.WithMaxStepsPerStory(10),
		ai.WithMaxStepsTotal(3),
	)
	report, err := a.Run(context.Background(), stories)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if report.Skipped == 0 {
		t.Errorf("expected at least 1 skipped story, got 0 (passed=%d, failed=%d)", report.Passed, report.Failed)
	}
}
