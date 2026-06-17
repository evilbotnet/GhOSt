package ai

// MCP (Model Context Protocol) client support. Ghost can use tools exposed by
// MCP servers the user configures in ~/.config/ghost/ai.toml. Each MCP tool is
// surfaced as a regular Ghost tool (namespaced "mcp__<server>__<tool>") and is
// confirmation-gated exactly like every other mutating action — see ghost.go.
//
// This is a minimal, dependency-free stdio JSON-RPC 2.0 client (stdlib + the
// already-present BurntSushi/toml). We deliberately do NOT pull in the official
// Go SDK: the wire protocol is small and stable, a self-contained file keeps the
// daemon's dependency surface tiny, and it covers the common case (npx-based
// stdio servers) completely. Streamable-HTTP transport is a documented TODO
// below.
//
// Design mirrors exttools.go: scan config -> build map[string]tool, ~16KB
// output cap, context timeouts, and never let a dead server break the loop.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	mcpProtocolVersion = "2025-06-18"
	mcpConnectTimeout  = 10 * time.Second
	mcpCallTimeout     = 60 * time.Second
	mcpOutputCap       = 16000
)

// --- config -----------------------------------------------------------------

// mcpServerConfig is one [[ai.mcp_servers]] entry. We re-decode ai.toml here
// rather than touching config.go, so MCP config evolves independently.
type mcpServerConfig struct {
	Name      string   `toml:"name"`
	Transport string   `toml:"transport"` // "stdio" (default) or "http"
	Command   []string `toml:"command"`   // argv, for stdio
	URL       string   `toml:"url"`       // for http (streamable) — TODO
	Enabled   bool     `toml:"enabled"`
}

type mcpConfigFile struct {
	AI struct {
		MCPServers []mcpServerConfig `toml:"mcp_servers"`
	} `toml:"ai"`
}

// loadMCPServers reads enabled MCP server configs from ai.toml. Safe (returns
// nil) when the file or section is absent.
func loadMCPServers() []mcpServerConfig {
	var f mcpConfigFile
	if _, err := toml.DecodeFile(ConfigPath(), &f); err != nil {
		return nil
	}
	var out []mcpServerConfig
	for _, s := range f.AI.MCPServers {
		if !s.Enabled || s.Name == "" {
			continue
		}
		if s.Transport == "" {
			s.Transport = "stdio"
		}
		out = append(out, s)
	}
	return out
}

// --- public surface ---------------------------------------------------------

// MCPServerInfo is the per-server listing for the shell to display.
type MCPServerInfo struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	ToolCount int    `json:"toolCount"`
	Error     string `json:"error,omitempty"`
}

// MCPServersInfo reports the live state of every configured MCP server. It
// connects lazily through the shared manager, so the first call also primes the
// connections. Safe to call with no servers configured (returns empty slice).
func MCPServersInfo() []MCPServerInfo {
	servers := loadMCPServers()
	out := make([]MCPServerInfo, 0, len(servers))
	for _, sc := range servers {
		conn, err := mgr.get(sc)
		info := MCPServerInfo{Name: sc.Name}
		if err != nil {
			info.Error = err.Error()
			out = append(out, info)
			continue
		}
		info.Connected = true
		info.ToolCount = len(conn.tools)
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// mcpTools builds runnable tool entries for the agent loop, one per tool across
// all enabled MCP servers. Connections are cached and reused (see manager); a
// dead/misconfigured server is skipped and surfaced via MCPServersInfo rather
// than failing the loop. Safe with no servers configured (returns empty map).
func mcpTools() map[string]tool {
	tools := map[string]tool{}
	for _, sc := range loadMCPServers() {
		conn, err := mgr.get(sc)
		if err != nil {
			log.Printf("ai/mcp: server %q unavailable: %v", sc.Name, err)
			continue
		}
		server := sc.Name
		for _, mt := range conn.tools {
			mt := mt
			name := "mcp__" + server + "__" + mt.Name
			props, required := schemaFields(mt.InputSchema)
			tools[name] = tool{
				def: ToolDef{
					Name:        name,
					Description: "[" + server + "] " + mt.Description,
					Properties:  props,
					Required:    required,
				},
				// Mutating unless the tool explicitly declares it read-only.
				// "mutating" only means "confirmation-gated", so defaulting to
				// true is the safe choice when the hint is absent.
				mutating: !mt.Annotations.ReadOnlyHint,
				run: func(args map[string]any) (string, error) {
					return mgr.callTool(server, mt.Name, args)
				},
			}
		}
	}
	return tools
}

// schemaFields pulls properties/required out of an MCP inputSchema (a JSON
// Schema object) into the shapes ToolDef expects.
func schemaFields(schema map[string]any) (props map[string]any, required []string) {
	if schema == nil {
		return map[string]any{}, nil
	}
	if p, ok := schema["properties"].(map[string]any); ok {
		props = p
	} else {
		props = map[string]any{}
	}
	if r, ok := schema["required"].([]any); ok {
		for _, v := range r {
			if s, ok := v.(string); ok {
				required = append(required, s)
			}
		}
	}
	return props, required
}

// --- manager (cached connections) -------------------------------------------

type mcpManager struct {
	mu    sync.Mutex
	conns map[string]*mcpConn // keyed by server name
}

var mgr = &mcpManager{conns: map[string]*mcpConn{}}

// get returns a live connection for the server, reconnecting if the cached one
// has died. Reconnection means we never spawn a fresh subprocess per tool call
// in the steady state, but a crashed server self-heals on the next use.
func (m *mcpManager) get(sc mcpServerConfig) (*mcpConn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c := m.conns[sc.Name]; c != nil {
		if c.alive() {
			return c, nil
		}
		c.close() // dead — drop and reconnect below
		delete(m.conns, sc.Name)
	}

	c, err := dialMCP(sc)
	if err != nil {
		return nil, err
	}
	m.conns[sc.Name] = c
	return c, nil
}

// callTool invokes a tool on the named server, transparently reconnecting once
// if the connection has died since it was cached.
func (m *mcpManager) callTool(server, toolName string, args map[string]any) (string, error) {
	m.mu.Lock()
	c := m.conns[server]
	m.mu.Unlock()
	if c == nil || !c.alive() {
		// Re-dial using current config.
		var sc *mcpServerConfig
		for _, s := range loadMCPServers() {
			if s.Name == server {
				s := s
				sc = &s
				break
			}
		}
		if sc == nil {
			return "", fmt.Errorf("mcp server %q is no longer configured", server)
		}
		var err error
		c, err = m.get(*sc)
		if err != nil {
			return "", err
		}
	}
	return c.callTool(toolName, args)
}

// --- connection (stdio JSON-RPC 2.0) ----------------------------------------

type mcpConn struct {
	server string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc

	tools []mcpToolDef

	writeMu sync.Mutex
	nextID  int64

	mu      sync.Mutex
	pending map[int64]chan rpcResponse

	dead atomic.Bool
}

type mcpToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations struct {
		ReadOnlyHint bool `json:"readOnlyHint"`
	} `json:"annotations"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message) }

// dialMCP launches the server, performs the initialize handshake, sends the
// initialized notification, and lists tools. Only stdio is implemented; http is
// a documented TODO.
func dialMCP(sc mcpServerConfig) (*mcpConn, error) {
	if sc.Transport == "http" {
		// TODO: Streamable-HTTP transport. The npx/stdio path covers the
		// common case (filesystem, everything, git, etc. servers); HTTP needs
		// an SSE/streamable POST loop that's out of scope for this minimal
		// client. Surface as an error so it shows up in MCPServersInfo.
		return nil, fmt.Errorf("http transport not yet supported for server %q", sc.Name)
	}
	if len(sc.Command) == 0 {
		return nil, fmt.Errorf("stdio server %q has no command", sc.Name)
	}

	// Long-lived process; we manage its lifetime with our own cancel.
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, sc.Command[0], sc.Command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = nil // server diagnostics are not our problem; swallow.
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start %q: %w", sc.Name, err)
	}

	c := &mcpConn{
		server:  sc.Name,
		cmd:     cmd,
		stdin:   stdin,
		cancel:  cancel,
		pending: map[int64]chan rpcResponse{},
	}
	go c.readLoop(stdout)
	go func() { _ = cmd.Wait(); c.markDead() }()

	if err := c.handshake(); err != nil {
		c.close()
		return nil, err
	}
	tools, err := c.listTools()
	if err != nil {
		c.close()
		return nil, err
	}
	c.tools = tools
	return c, nil
}

func (c *mcpConn) markDead() {
	c.dead.Store(true)
	// Fail any in-flight callers so they don't block.
	c.mu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

func (c *mcpConn) alive() bool { return !c.dead.Load() }

func (c *mcpConn) close() {
	c.dead.Store(true)
	if c.cancel != nil {
		c.cancel()
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
}

// readLoop reads newline-delimited JSON-RPC messages from the server and routes
// responses to their waiting callers. Notifications/requests from the server
// (it has no reason to call us with the tools we use) are ignored.
func (c *mcpConn) readLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.ID == nil {
			continue // notification or non-id message — ignore
		}
		c.mu.Lock()
		ch := c.pending[*resp.ID]
		delete(c.pending, *resp.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- resp
			close(ch)
		}
	}
	c.markDead()
}

// call sends a request and waits for the matching response (or timeout).
func (c *mcpConn) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if !c.alive() {
		return nil, fmt.Errorf("mcp server %q is not running", c.server)
	}
	id := atomic.AddInt64(&c.nextID, 1)
	ch := make(chan rpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := rpcRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params}
	if err := c.write(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("mcp server %q died during %q", c.server, method)
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("mcp %q timed out: %w", method, ctx.Err())
	}
}

// notify sends a fire-and-forget JSON-RPC notification (no id, no response).
func (c *mcpConn) notify(method string, params any) error {
	return c.write(rpcRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (c *mcpConn) write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := c.stdin.Write(data); err != nil {
		c.markDead()
		return err
	}
	return nil
}

// handshake performs initialize + notifications/initialized.
func (c *mcpConn) handshake() error {
	ctx, cancel := context.WithTimeout(context.Background(), mcpConnectTimeout)
	defer cancel()
	params := map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "ghost", "version": "1.0"},
	}
	if _, err := c.call(ctx, "initialize", params); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if err := c.notify("notifications/initialized", map[string]any{}); err != nil {
		return fmt.Errorf("initialized: %w", err)
	}
	return nil
}

func (c *mcpConn) listTools() ([]mcpToolDef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mcpConnectTimeout)
	defer cancel()
	raw, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}
	var res struct {
		Tools []mcpToolDef `json:"tools"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("tools/list decode: %w", err)
	}
	return res.Tools, nil
}

// callTool invokes one tool and returns its text content joined. Non-text
// content blocks are summarized rather than dumped.
func (c *mcpConn) callTool(toolName string, args map[string]any) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mcpCallTimeout)
	defer cancel()
	if args == nil {
		args = map[string]any{}
	}
	raw, err := c.call(ctx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}

	var res struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("tools/call decode: %w", err)
	}

	var parts []string
	for _, blk := range res.Content {
		if blk.Type == "text" {
			parts = append(parts, blk.Text)
		} else {
			parts = append(parts, fmt.Sprintf("(%s content omitted)", blk.Type))
		}
	}
	out := strings.TrimSpace(strings.Join(parts, "\n"))
	if len(out) > mcpOutputCap {
		out = out[:mcpOutputCap] + "\n…(truncated)"
	}
	if out == "" {
		out = "(no output)"
	}
	if res.IsError {
		// The MCP spec carries tool execution errors in the result with
		// isError=true; bubble it up as a Go error so the loop marks it.
		return "", fmt.Errorf("%s", out)
	}
	return out, nil
}
