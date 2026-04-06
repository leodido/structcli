package mcp

import "encoding/json"

// ProtocolVersion is the MCP protocol version reported by SetupMCP.
const ProtocolVersion = "2024-11-05"

// Options configures the --mcp flag for command-line applications.
type Options struct {
	FlagName    string   // Name of the persistent flag (defaults to "mcp")
	Name        string   // Server name reported during initialize (defaults to root command name)
	Version     string   // Server version reported during initialize (defaults to structcli.Version)
	Separator   string   // Tool name separator for nested commands (defaults to "-")
	AllCommands bool     // Include runnable parent/root commands, not just leaf commands
	Exclude     []string // Exclude tool names or full command paths from tools/list and tools/call
}

// Request is a JSON-RPC request sent over MCP stdio.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response sent over MCP stdio.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is a JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeResult is returned from the initialize request.
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
}

// ServerInfo describes the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool is exposed by tools/list.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult is returned from tools/list.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams are provided to tools/call.
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolCallContent is a single text content part in a tools/call result.
type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolCallResult is returned from tools/call.
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}
