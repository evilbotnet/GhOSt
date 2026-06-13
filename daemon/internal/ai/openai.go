package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// openaiLLM speaks the OpenAI-compatible chat/completions dialect — the
// lingua franca of Ollama, vLLM, llama.cpp --server, and most proxies.
type openaiLLM struct {
	url   string
	model string
	key   string
	http  *http.Client
}

func newOpenAI(p Provider) LLM {
	return &openaiLLM{
		url:   strings.TrimRight(p.URL, "/"),
		model: p.Model,
		key:   p.Key(),
		http:  &http.Client{Timeout: 180 * time.Second},
	}
}

func (o *openaiLLM) Chat(ctx context.Context, system string, msgs []Msg, tools []ToolDef) (Turn, error) {
	type oaFunc struct {
		Name      string         `json:"name"`
		Arguments string         `json:"arguments,omitempty"`
		Desc      string         `json:"description,omitempty"`
		Params    map[string]any `json:"parameters,omitempty"`
	}
	type oaToolCall struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	type oaMsg struct {
		Role       string       `json:"role"`
		Content    string       `json:"content"`
		ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
		ToolCallID string       `json:"tool_call_id,omitempty"`
	}

	oms := []oaMsg{{Role: "system", Content: system}}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			oms = append(oms, oaMsg{Role: "user", Content: m.Text})
		case "assistant":
			om := oaMsg{Role: "assistant", Content: m.Text}
			for _, c := range m.Calls {
				args, _ := json.Marshal(c.Args)
				tc := oaToolCall{ID: c.ID, Type: "function"}
				tc.Function.Name = c.Name
				tc.Function.Arguments = string(args)
				om.ToolCalls = append(om.ToolCalls, tc)
			}
			oms = append(oms, om)
		case "tool":
			for _, r := range m.Results {
				oms = append(oms, oaMsg{Role: "tool", Content: r.Content, ToolCallID: r.CallID})
			}
		}
	}

	var toolsField []map[string]any
	for _, t := range tools {
		toolsField = append(toolsField, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters": map[string]any{
					"type": "object", "properties": t.Properties, "required": t.Required,
				},
			},
		})
	}

	body, _ := json.Marshal(map[string]any{
		"model": o.model, "messages": oms, "tools": toolsField,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", o.url+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Turn{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.key != "" {
		req.Header.Set("Authorization", "Bearer "+o.key)
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return Turn{}, fmt.Errorf("endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != 200 {
		return Turn{}, fmt.Errorf("endpoint returned %d: %.200s", resp.StatusCode, data)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content   string       `json:"content"`
				ToolCalls []oaToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil || len(out.Choices) == 0 {
		return Turn{}, fmt.Errorf("bad response from endpoint")
	}
	msg := out.Choices[0].Message
	turn := Turn{Text: msg.Content}
	for i, tc := range msg.ToolCalls {
		args := map[string]any{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		id := tc.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		turn.Calls = append(turn.Calls, ToolCall{ID: id, Name: tc.Function.Name, Args: args})
	}
	return turn, nil
}
