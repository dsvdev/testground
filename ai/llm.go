package ai

import (
	"context"
	"encoding/json"
)

type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema (type=object)
}

type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type Message struct {
	Role       string // "user" | "assistant" | "tool"
	Content    string
	ToolCalls  []ToolCall // filled if Role="assistant" and LLM called tools
	ToolCallID string     // filled if Role="tool"
}

type Response struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool // true if no tool_calls (LLM answered with text only)
}

type LLMClient interface {
	Complete(ctx context.Context, messages []Message, tools []Tool) (*Response, error)
}
