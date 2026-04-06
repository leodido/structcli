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

type failingFlagValue struct {
	value string
}

func (f *failingFlagValue) String() string { return f.value }

func (f *failingFlagValue) Set(value string) error {
	if value == "default" {
		return fmt.Errorf("cannot reset to %q", value)
	}
	f.value = value
	return nil
}

func (f *failingFlagValue) Type() string { return "failing" }

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

func TestSetupMCP_PreservesPersistentPreRun(t *testing.T) {
	called := false
	root := &cobra.Command{
		Use: "myapp",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			called = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, SetupMCP(root, structclimcp.Options{}))

	root.SetArgs(nil)
	require.NoError(t, root.Execute())
	assert.True(t, called)
}

func TestSetupMCP_ExecuteInterceptsWithoutExit(t *testing.T) {
	root := newMCPLeafRoot(t)
	root.SilenceErrors = true
	root.SilenceUsage = true
	require.NoError(t, SetupMCP(root, structclimcp.Options{}))

	var out bytes.Buffer
	root.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"))
	root.SetOut(&out)
	root.SetArgs([]string{"srv", "--mcp"})
	require.NoError(t, root.Execute())

	var resp structclimcp.Response
	require.NoError(t, json.Unmarshal(out.Bytes(), &resp))
	var listResult structclimcp.ToolsListResult
	mustUnmarshalJSON(t, resp.Result, &listResult)
	assert.Contains(t, toolNames(listResult.Tools), "srv")
	assert.NotContains(t, out.String(), "started")

	out.Reset()
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"srv"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "port" not set`)
}

func TestSetupMCP_InterceptsRootHooksAssignedAfterSetup(t *testing.T) {
	root := &cobra.Command{Use: "myapp", SilenceErrors: true, SilenceUsage: true}
	require.NoError(t, SetupMCP(root, structclimcp.Options{}))

	opts := &mcpLeafOptions{}
	require.NoError(t, opts.Attach(root))

	preRunCalls := 0
	runCalls := 0
	root.PreRunE = func(c *cobra.Command, args []string) error {
		preRunCalls++
		return Unmarshal(c, opts)
	}
	root.RunE = func(c *cobra.Command, args []string) error {
		runCalls++
		fmt.Fprint(c.OutOrStdout(), "started")
		return nil
	}

	var out bytes.Buffer
	root.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"))
	root.SetOut(&out)
	root.SetArgs([]string{"--mcp"})
	require.NoError(t, root.Execute())

	var resp structclimcp.Response
	require.NoError(t, json.Unmarshal(out.Bytes(), &resp))
	var listResult structclimcp.ToolsListResult
	mustUnmarshalJSON(t, resp.Result, &listResult)
	assert.Contains(t, toolNames(listResult.Tools), "myapp")
	assert.Zero(t, preRunCalls)
	assert.Zero(t, runCalls)
	assert.NotContains(t, out.String(), "started")

	out.Reset()
	root.SetIn(strings.NewReader(""))
	root.SetArgs(nil)
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "port" not set`)
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

func TestHandleMCPRequest_ErrorResponses(t *testing.T) {
	root := newMCPRunnableParentRoot(t)
	cfg := resolveMCPConfig(root, structclimcp.Options{AllCommands: true})
	registry, err := newMCPRegistry(root, cfg)
	require.NoError(t, err)

	t.Run("nil request", func(t *testing.T) {
		resp, err := handleMCPRequest(root, cfg, registry, nil)
		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid jsonrpc", func(t *testing.T) {
		resp, err := handleMCPRequest(root, cfg, registry, &structclimcp.Request{
			JSONRPC: "1.0",
			ID:      json.RawMessage(`1`),
			Method:  "tools/list",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, rpcCodeInvalidRequest, resp.Error.Code)
		assert.Equal(t, "jsonrpc must be 2.0", resp.Error.Message)
	})

	t.Run("unknown method", func(t *testing.T) {
		resp, err := handleMCPRequest(root, cfg, registry, &structclimcp.Request{
			JSONRPC: jsonrpcVersion,
			ID:      json.RawMessage(`2`),
			Method:  "tools/missing",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, rpcCodeMethodNotFound, resp.Error.Code)
	})

	t.Run("notification without id is ignored", func(t *testing.T) {
		resp, err := handleMCPRequest(root, cfg, registry, &structclimcp.Request{
			JSONRPC: jsonrpcVersion,
			Method:  "tools/missing",
		})
		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid tools call params", func(t *testing.T) {
		resp, err := handleMCPRequest(root, cfg, registry, &structclimcp.Request{
			JSONRPC: jsonrpcVersion,
			ID:      json.RawMessage(`3`),
			Method:  "tools/call",
			Params:  json.RawMessage(`{`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, rpcCodeInvalidParams, resp.Error.Code)
		assert.Equal(t, "invalid tools/call params", resp.Error.Message)
	})

	t.Run("missing tool name", func(t *testing.T) {
		params, err := json.Marshal(map[string]any{
			"arguments": map[string]any{"port": 3000},
		})
		require.NoError(t, err)

		resp, err := handleMCPRequest(root, cfg, registry, &structclimcp.Request{
			JSONRPC: jsonrpcVersion,
			ID:      json.RawMessage(`4`),
			Method:  "tools/call",
			Params:  params,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, rpcCodeInvalidParams, resp.Error.Code)
		assert.Equal(t, "tool name is required", resp.Error.Message)
	})
}

func TestMCPArgumentHelpers(t *testing.T) {
	t.Run("arguments to argv", func(t *testing.T) {
		schema := &CommandSchema{
			Flags: map[string]*FlagSchema{
				"host": {},
				"port": {},
				"tags": {},
			},
		}

		args, err := mcpArgumentsToArgs(schema, map[string]any{
			"host": "0.0.0.0",
			"port": json.Number("3000"),
			"tags": []any{"alpha", true, float64(1.5)},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{
			"--host", "0.0.0.0",
			"--port", "3000",
			"--tags", "alpha",
			"--tags", "true",
			"--tags", "1.5",
		}, args)
	})

	t.Run("unknown argument", func(t *testing.T) {
		schema := &CommandSchema{Flags: map[string]*FlagSchema{"host": {}}}

		args, err := mcpArgumentsToArgs(schema, map[string]any{"port": 3000})
		require.Error(t, err)
		assert.Nil(t, args)
		assert.EqualError(t, err, `unknown argument "port"`)
	})

	t.Run("invalid repeated argument value", func(t *testing.T) {
		schema := &CommandSchema{Flags: map[string]*FlagSchema{"tags": {}}}

		args, err := mcpArgumentsToArgs(schema, map[string]any{
			"tags": []any{make(chan int)},
		})
		require.Error(t, err)
		assert.Nil(t, args)
		assert.Contains(t, err.Error(), `invalid argument "tags"`)
	})

	t.Run("stringify supported values", func(t *testing.T) {
		values, err := mcpArgumentValues([]any{
			"hello",
			true,
			json.Number("42"),
			float32(1.25),
			int64(-7),
			uint(9),
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"hello", "true", "42", "1.25", "-7", "9"}, values)

		value, err := mcpArgumentString([]int{1, 2})
		require.NoError(t, err)
		assert.Equal(t, "[1,2]", value)
	})

	t.Run("unsupported scalar value", func(t *testing.T) {
		value, err := mcpArgumentString(make(chan int))
		require.Error(t, err)
		assert.Empty(t, value)
		assert.Contains(t, err.Error(), "unsupported value type chan int")
	})
}

func TestResetCommandExecutionState_ReturnsFlagResetError(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	value := &failingFlagValue{value: "default"}
	root.Flags().Var(value, "broken", "broken flag")

	require.NoError(t, root.Flags().Set("broken", "custom"))
	assert.True(t, root.Flags().Lookup("broken").Changed)

	err := resetCommandExecutionState(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resetting flag broken")
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
