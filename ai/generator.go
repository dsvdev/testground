package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func Generate(ctx context.Context, llm LLMClient, project *ProjectContext) ([]UserStory, error) {
	prompt := buildPrompt(project)
	messages := []Message{{Role: "user", Content: prompt}}

	resp, err := llm.Complete(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	stories, err := parseStories(resp.Content)
	if err != nil {
		messages = append(messages,
			Message{Role: "assistant", Content: resp.Content},
			Message{Role: "user", Content: "Return only JSON array, your previous response was not valid JSON"},
		)
		resp2, err2 := llm.Complete(ctx, messages, nil)
		if err2 != nil {
			return nil, fmt.Errorf("llm complete retry: %w", err2)
		}
		return parseStories(resp2.Content)
	}
	return stories, nil
}

func buildPrompt(project *ProjectContext) string {
	var sb strings.Builder

	sb.WriteString("You are an integration test planner for a Go backend service.\n\n")

	sb.WriteString("Project endpoints:\n")
	if len(project.Endpoints) == 0 {
		sb.WriteString("  none\n")
	}
	for _, ep := range project.Endpoints {
		sb.WriteString(fmt.Sprintf("  %s %s (handler: %s)\n", ep.Method, ep.Path, ep.Handler))
	}

	sb.WriteString("\nData models:\n")
	if len(project.Models) == 0 {
		sb.WriteString("  none\n")
	}
	for _, m := range project.Models {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", m.Name, strings.Join(m.Fields, ", ")))
	}

	tables := "none"
	if len(project.Tables) > 0 {
		tables = strings.Join(project.Tables, ", ")
	}
	sb.WriteString(fmt.Sprintf("\nDatabase tables: %s\n", tables))

	topics := "none"
	if len(project.Topics) > 0 {
		topics = strings.Join(project.Topics, ", ")
	}
	sb.WriteString(fmt.Sprintf("Kafka topics: %s\n", topics))

	sb.WriteString(`
Generate a JSON array of user stories covering:
- Main happy path for each endpoint
- Key error cases (404, invalid input)
- Cross-service flows (create entity → verify side effect in DB or Kafka)

Each story:
{
  "title": "short name",
  "description": "what is being tested",
  "steps": ["human-readable step 1", "step 2", ...]
}

Return ONLY valid JSON array. No markdown, no explanation.`)

	return sb.String()
}

type storyJSON struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
}

func parseStories(content string) ([]UserStory, error) {
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	var raw []storyJSON
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("parse stories: %w", err)
	}

	stories := make([]UserStory, len(raw))
	for i, r := range raw {
		stories[i] = UserStory{
			Title:       r.Title,
			Description: r.Description,
			Steps:       r.Steps,
		}
	}
	return stories, nil
}
