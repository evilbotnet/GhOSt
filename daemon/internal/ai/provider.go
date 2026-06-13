package ai

import "context"

// The provider-agnostic conversation model. Both providers translate to/from
// this; the loop in session.go only sees these types.

type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type ToolResult struct {
	CallID  string `json:"callId"`
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

type Msg struct {
	Role    string // "user" | "assistant" | "tool"
	Text    string
	Calls   []ToolCall   // assistant turns
	Results []ToolResult // tool turns
}

// Turn is one model response: optional text plus zero or more tool calls.
type Turn struct {
	Text  string
	Calls []ToolCall
}

type ToolDef struct {
	Name        string
	Description string
	Properties  map[string]any
	Required    []string
}

type LLM interface {
	Chat(ctx context.Context, system string, msgs []Msg, tools []ToolDef) (Turn, error)
}
