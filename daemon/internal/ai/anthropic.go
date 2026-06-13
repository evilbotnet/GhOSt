package ai

import (
	"context"
	"encoding/json"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

type anthropicLLM struct {
	client anthropic.Client
	model  string
}

func newAnthropic(p Provider) LLM {
	model := p.Model
	if model == "" {
		model = "claude-opus-4-8"
	}
	opts := []option.RequestOption{option.WithAPIKey(p.Key())}
	if p.URL != "" {
		opts = append(opts, option.WithBaseURL(p.URL))
	}
	return &anthropicLLM{client: anthropic.NewClient(opts...), model: model}
}

func (a *anthropicLLM) Chat(ctx context.Context, system string, msgs []Msg, tools []ToolDef) (Turn, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 4096,
		System:    []anthropic.TextBlockParam{{Text: system}},
	}
	for _, t := range tools {
		params.Tools = append(params.Tools, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: param.NewOpt(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: t.Properties, Required: t.Required},
		}})
	}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			params.Messages = append(params.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Text)))
		case "assistant":
			var blocks []anthropic.ContentBlockParamUnion
			if m.Text != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Text))
			}
			for _, c := range m.Calls {
				blocks = append(blocks, anthropic.NewToolUseBlock(c.ID, c.Args, c.Name))
			}
			params.Messages = append(params.Messages, anthropic.NewAssistantMessage(blocks...))
		case "tool":
			var blocks []anthropic.ContentBlockParamUnion
			for _, r := range m.Results {
				blocks = append(blocks, anthropic.NewToolResultBlock(r.CallID, r.Content, r.IsError))
			}
			params.Messages = append(params.Messages, anthropic.NewUserMessage(blocks...))
		}
	}

	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return Turn{}, fmt.Errorf("anthropic: %w", err)
	}
	var turn Turn
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			turn.Text += block.Text
		case "tool_use":
			args := map[string]any{}
			json.Unmarshal(block.Input, &args)
			turn.Calls = append(turn.Calls, ToolCall{ID: block.ID, Name: block.Name, Args: args})
		}
	}
	return turn, nil
}
