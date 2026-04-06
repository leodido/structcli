package structcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mcpLeafOptions struct {
	Host string `flag:"host" default:"localhost"`
	Port int    `flag:"port" flagrequired:"true"`
}

func (o *mcpLeafOptions) Attach(c *cobra.Command) error {
	return Define(c, o)
}

func newMCPLeafRoot(t *testing.T) *cobra.Command {
	t.Helper()

	root := &cobra.Command{
		Use:   "myapp",
		Short: "Test app",
	}
	SetupFlagErrors(root)

	opts := &mcpLeafOptions{}
	srv := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		PreRunE: func(c *cobra.Command, args []string) error {
			return Unmarshal(c, opts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(c.OutOrStdout(), "started %s:%d", opts.Host, opts.Port)
			return nil
		},
	}
	require.NoError(t, opts.Attach(srv))
	root.AddCommand(srv)

	return root
}

func newMCPRunnableParentRoot(t *testing.T) *cobra.Command {
	t.Helper()

	root := &cobra.Command{
		Use:   "myapp",
		Short: "Test app",
		Run: func(c *cobra.Command, args []string) {
			fmt.Fprint(c.OutOrStdout(), "root")
		},
	}
	root.Flags().Bool("dry", false, "Dry run")

	srv := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		Run: func(c *cobra.Command, args []string) {
			fmt.Fprint(c.OutOrStdout(), "srv")
		},
	}

	version := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprint(c.OutOrStdout(), "version")
			return nil
		},
	}
	srv.AddCommand(version)

	ping := &cobra.Command{
		Use:   "ping",
		Short: "Ping the service",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprint(c.OutOrStdout(), "pong")
			return nil
		},
	}

	root.AddCommand(srv, ping)
	return root
}

func TestSetupMCP_PreservesPersistentPreRunE(t *testing.T) {
	called := false
	root := &cobra.Command{
		Use: "myapp",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			called = true
			return nil
		},
	}

	require.NoError(t, SetupMCP(root, structclimcp.Options{}))
	require.NoError(t, root.PersistentPreRunE(root, nil))
	assert.True(t, called)
}

func TestJSONSchema_SkipsMCPFlag(t *testing.T) {
	root := newMCPLeafRoot(t)
	require.NoError(t, SetupMCP(root, structclimcp.Options{}))

	schemas, err := JSONSchema(root)
	require.NoError(t, err)
	require.Len(t, schemas, 1)
	assert.NotContains(t, schemas[0].Flags, "mcp")
}

func TestRunMCPServer_InitializeAndToolsList_DefaultLeafOnly(t *testing.T) {
	root := newMCPRunnableParentRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{}),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"test"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)

	require.Len(t, responses, 2)

	var initResult structclimcp.InitializeResult
	mustUnmarshalJSON(t, responses[0].Result, &initResult)
	assert.Equal(t, structclimcp.ProtocolVersion, initResult.ProtocolVersion)
	assert.Equal(t, "myapp", initResult.ServerInfo.Name)

	var listResult structclimcp.ToolsListResult
	mustUnmarshalJSON(t, responses[1].Result, &listResult)
	require.Len(t, listResult.Tools, 2)
	assert.Equal(t, []string{"ping", "srv-version"}, toolNames(listResult.Tools))
}

func TestRunMCPServer_ToolsList_AllCommands(t *testing.T) {
	root := newMCPRunnableParentRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{AllCommands: true}),
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
	)

	require.Len(t, responses, 1)

	var listResult structclimcp.ToolsListResult
	mustUnmarshalJSON(t, responses[0].Result, &listResult)
	assert.Equal(t, []string{"myapp", "ping", "srv", "srv-version"}, toolNames(listResult.Tools))
}

func TestRunMCPServer_ToolsList_Exclude(t *testing.T) {
	root := newMCPRunnableParentRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{
		AllCommands: true,
		Exclude:     []string{"ping", "myapp srv"},
	}),
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
	)

	require.Len(t, responses, 1)

	var listResult structclimcp.ToolsListResult
	mustUnmarshalJSON(t, responses[0].Result, &listResult)
	assert.Equal(t, []string{"myapp", "srv-version"}, toolNames(listResult.Tools))
}

func TestRunMCPServer_ToolsCallSuccess(t *testing.T) {
	root := newMCPLeafRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{}),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"srv","arguments":{"host":"0.0.0.0","port":3000}}}`,
	)

	require.Len(t, responses, 1)

	var result structclimcp.ToolCallResult
	mustUnmarshalJSON(t, responses[0].Result, &result)
	require.Len(t, result.Content, 1)
	assert.False(t, result.IsError)
	assert.Equal(t, "started 0.0.0.0:3000", result.Content[0].Text)
}

func TestRunMCPServer_ToolsCallStructuredError(t *testing.T) {
	root := newMCPLeafRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{}),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"srv","arguments":{"port":"abc"}}}`,
	)

	require.Len(t, responses, 1)

	var result structclimcp.ToolCallResult
	mustUnmarshalJSON(t, responses[0].Result, &result)
	require.Len(t, result.Content, 1)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, `"error":"invalid_flag_value"`)
	assert.Contains(t, result.Content[0].Text, `"flag":"port"`)
}

func TestRunMCPServer_NotificationsAreIgnored(t *testing.T) {
	root := newMCPRunnableParentRoot(t)
	responses := runMCPTestServer(t, root, resolveMCPConfig(root, structclimcp.Options{}),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	)

	assert.Empty(t, responses)
}

func runMCPTestServer(t *testing.T, root *cobra.Command, cfg *mcpConfig, requests ...string) []structclimcp.Response {
	t.Helper()

	in := strings.NewReader(strings.Join(requests, "\n"))
	var out bytes.Buffer

	require.NoError(t, runMCPServer(root, cfg, in, &out))

	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	var responses []structclimcp.Response
	for {
		var resp structclimcp.Response
		err := dec.Decode(&resp)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		responses = append(responses, resp)
	}

	return responses
}

func mustUnmarshalJSON(t *testing.T, value any, target any) {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, target))
}

func toolNames(tools []structclimcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}
