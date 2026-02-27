package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RunConfig controls step limits for a single story execution.
type RunConfig struct {
	MaxStepsPerStory int
	StepsRemaining   int
}

type storyResult struct {
	story UserStory
	steps int
}

func runStory(ctx context.Context, llm LLMClient, executor *Executor, tools []Tool,
	story UserStory, cfg RunConfig, obs Observer, index, total int) storyResult {

	effectiveMax := cfg.MaxStepsPerStory
	if cfg.StepsRemaining < effectiveMax {
		effectiveMax = cfg.StepsRemaining
	}

	obs.OnStoryStart(index, total, story)

	prompt := buildRunPrompt(executor, story)
	messages := []Message{{Role: "user", Content: prompt}}

	start := time.Now()
	steps := 0
	done := false
	status := "failed"
	errMsg := "maxStepsPerStory reached"
	var stepResults []StepResult

	for steps < effectiveMax {
		resp, err := llm.Complete(ctx, messages, tools)
		if err != nil {
			story.Status = "failed"
			story.Error = fmt.Sprintf("llm error: %v", err)
			story.Duration = time.Since(start)
			story.StepResults = stepResults
			obs.OnStoryDone(index, total, story)
			return storyResult{story, steps}
		}

		if resp.Done {
			status, errMsg = parseRunResult(resp.Content)
			done = true
			break
		}

		if len(resp.ToolCalls) == 0 {
			// Guard: LLM returned neither text nor tool calls
			steps++
			continue
		}

		// Append assistant message with tool calls
		messages = append(messages, Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call and collect results
		for _, tc := range resp.ToolCalls {
			toolName, input, output := executor.Execute(ctx, tc)
			sr := StepResult{Tool: toolName, Input: input, Output: output}
			stepResults = append(stepResults, sr)
			obs.OnStep(index, total, sr)
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    output,
			})
		}

		steps += len(resp.ToolCalls)
	}

	if !done {
		status = "failed"
		errMsg = "maxStepsPerStory reached"
	}

	story.StepResults = stepResults
	story.Status = status
	story.Error = errMsg
	story.Duration = time.Since(start)
	obs.OnStoryDone(index, total, story)
	return storyResult{story, steps}
}

func buildRunPrompt(executor *Executor, story UserStory) string {
	var sb strings.Builder

	sb.WriteString("You are an integration test executor for a Go backend service.\n")
	if executor.serviceURL != "" {
		sb.WriteString(fmt.Sprintf("Service URL: %s\n", executor.serviceURL))
	}
	sb.WriteString("\nExecute this user story using the available tools:\n\n")
	sb.WriteString(fmt.Sprintf("Title: %s\n", story.Title))
	sb.WriteString(fmt.Sprintf("Description: %s\n", story.Description))

	if len(story.Steps) > 0 {
		sb.WriteString("\nExpected steps:\n")
		for i, step := range story.Steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	sb.WriteString(`
Rules:
- Start with sql_exec TRUNCATE to ensure clean state (if SQL tools are available)
- Execute each step using tools
- Verify HTTP response status after each request
- If any step fails, stop and report failure with clear reason
- When all steps complete, respond with exactly:
  PASSED
  or
  FAILED: <reason>
`)

	return sb.String()
}

func parseRunResult(content string) (status, errMsg string) {
	upper := strings.ToUpper(content)

	// Look for FAILED: first — it's more specific than PASSED.
	if idx := strings.Index(upper, "FAILED:"); idx != -1 {
		// Extract the reason: everything after "FAILED:" on the same line.
		rest := content[idx+len("FAILED:"):]
		if nl := strings.IndexByte(rest, '\n'); nl != -1 {
			rest = rest[:nl]
		}
		return "failed", strings.TrimSpace(rest)
	}

	if strings.Contains(upper, "PASSED") {
		return "passed", ""
	}

	return "failed", strings.TrimSpace(content)
}
