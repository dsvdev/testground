package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/dsvdev/testground/ai"
)

type AnthropicOption func(*AnthropicClient)

func WithModel(model string) AnthropicOption {
	return func(c *AnthropicClient) {
		c.model = model
	}
}

type AnthropicClient struct {
	client anthropic.Client
	model  string
}

func NewAnthropic(apiKey string, opts ...AnthropicOption) *AnthropicClient {
	c := &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  string(anthropic.ModelClaudeOpus4_5),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *AnthropicClient) Complete(ctx context.Context, messages []ai.Message, tools []ai.Tool) (*ai.Response, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		Messages:  convertMessages(messages),
	}
	if len(tools) > 0 {
		params.Tools = convertTools(tools)
	}

	msg, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic complete: %w", err)
	}

	var textSB strings.Builder
	var toolCalls []ai.ToolCall

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			textSB.WriteString(block.Text)
		case "tool_use":
			toolCalls = append(toolCalls, ai.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input, // json.RawMessage
			})
		}
	}

	return &ai.Response{
		Content:   textSB.String(),
		ToolCalls: toolCalls,
		Done:      len(toolCalls) == 0,
	}, nil
}

func convertMessages(messages []ai.Message) []anthropic.MessageParam {
	var result []anthropic.MessageParam
	i := 0
	for i < len(messages) {
		m := messages[i]
		switch m.Role {
		case "assistant":
			var blocks []anthropic.ContentBlockParamUnion
			if m.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Content))
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Input, tc.Name))
			}
			result = append(result, anthropic.NewAssistantMessage(blocks...))
			i++
		case "tool":
			// Group consecutive "tool" messages into a single user message
			var toolBlocks []anthropic.ContentBlockParamUnion
			for i < len(messages) && messages[i].Role == "tool" {
				toolBlocks = append(toolBlocks,
					anthropic.NewToolResultBlock(messages[i].ToolCallID, messages[i].Content, false),
				)
				i++
			}
			result = append(result, anthropic.NewUserMessage(toolBlocks...))
		default: // "user"
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
			i++
		}
	}
	return result
}

func convertTools(tools []ai.Tool) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var schema struct {
			Properties any      `json:"properties"`
			Required   []string `json:"required"`
		}
		json.Unmarshal(t.InputSchema, &schema) //nolint:errcheck

		tp := anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Properties: schema.Properties,
				Required:   schema.Required,
			},
			t.Name,
		)
		tp.OfTool.Description = param.NewOpt(t.Description)
		result = append(result, tp)
	}
	return result
}