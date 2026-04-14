package mcp

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
)

// ProtocolVersion is the MCP protocol version reported by SetupMCP.
const ProtocolVersion = "2024-11-05"

// CommandFactory builds a fresh Cobra command tree for a single MCP tools/call.
//
// Use this when a CLI captures output streams into closures at construction
// time, or when reusing and resetting the same Cobra tree is unsafe.
//
// Parameters:
//   - argv: the command path and flag arguments (excluding the root command
//     name). Provided for informational use or conditional subtree building.
//     The factory must not call SetArgs — the caller handles that.
//   - stdout, stderr: writers that must receive all command output. Wire these
//     into any closures that capture output streams at construction time.
//
// After the factory returns, the caller configures the returned command:
// SetArgs, SetIn, SetOut, SetErr, SilenceErrors, and SilenceUsage. The
// factory must not set these. The caller's SetOut/SetErr use the same
// buffers passed as stdout/stderr, ensuring cmd.OutOrStdout() works
// alongside closure-captured writers.
//
// The returned command tree must have the same structure and flags as the
// root command passed to [SetupMCP]. The MCP tool registry is built from
// the original tree; the factory tree is only used for execution.
type CommandFactory func(argv []string, stdout io.Writer, stderr io.Writer) (*cobra.Command, error)

// Options configures the --mcp flag for command-line applications.
type Options struct {
	FlagName       string         // Name of the persistent flag (defaults to "mcp")
	Name           string         // Server name reported during initialize (defaults to root command name)
	Version        string         // Server version reported during initialize (defaults to structcli.Version)
	Separator      string         // Tool name separator for nested commands (defaults to "-")
	AllCommands    bool           // Include runnable parent/root commands. By default MCP exposes runnable leaves only.
	Exclude        []string       // Exclude tool names or full command paths from tools/list and tools/call
	CommandFactory CommandFactory // Optional fresh command factory for each MCP tools/call execution.
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
