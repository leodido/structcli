package structcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	internalcmd "github.com/leodido/structcli/internal/cmd"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	mcpFlagAnnotation = "leodido/structcli/mcp-flag"

	jsonrpcVersion = "2.0"

	rpcCodeParseError     = -32700
	rpcCodeInvalidRequest = -32600
	rpcCodeMethodNotFound = -32601
	rpcCodeInvalidParams  = -32602
	rpcCodeInternalError  = -32603
)

type mcpConfig struct {
	name           string
	version        string
	flagName       string
	separator      string
	allCommands    bool
	exclude        map[string]struct{}
	commandFactory structclimcp.CommandFactory
}

type mcpToolDef struct {
	name   string
	schema *CommandSchema
	path   []string
}

type mcpRegistry struct {
	tools []structclimcp.Tool
	defs  map[string]*mcpToolDef
}

// SetupMCP adds a --mcp persistent flag to the root command.
//
// When the flag is set, the command serves a minimal MCP server over stdio and
// returns without running the command's normal execution path.
// Works only for the root command.
func SetupMCP(rootC *cobra.Command, opts structclimcp.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupMCP must be called on the root command")
	}

	cfg := resolveMCPConfig(rootC, opts)

	rootC.PersistentFlags().Bool(cfg.flagName, false, "serve MCP over stdio")

	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	rootC.Annotations[mcpFlagAnnotation] = cfg.flagName

	internalcmd.EnsureRunnable(rootC)

	// Wrap right before execution so commands and hooks added after setup are
	// still intercepted before Cobra validates args and required flags.
	cobra.OnInitialize(func() {
		wrapForMCP(rootC, cfg)
	})
	SetupUsage(rootC)

	return nil
}

func resolveMCPConfig(rootC *cobra.Command, opts structclimcp.Options) *mcpConfig {
	cfg := &mcpConfig{
		name:           opts.Name,
		version:        opts.Version,
		flagName:       opts.FlagName,
		separator:      opts.Separator,
		allCommands:    opts.AllCommands,
		exclude:        make(map[string]struct{}, len(opts.Exclude)),
		commandFactory: opts.CommandFactory,
	}
	if cfg.flagName == "" {
		cfg.flagName = "mcp"
	}
	if cfg.name == "" {
		cfg.name = rootC.Name()
	}
	if cfg.version == "" {
		cfg.version = Version
	}
	if cfg.separator == "" {
		cfg.separator = "-"
	}
	for _, item := range opts.Exclude {
		if item == "" {
			continue
		}
		cfg.exclude[item] = struct{}{}
	}

	return cfg
}

func wrapForMCP(rootC *cobra.Command, cfg *mcpConfig) {
	internalcmd.RecursivelyWrapExecution(rootC, internalcmd.ExecutionInterceptor{
		Annotation: "leodido/structcli/mcp-wrapped",
		ShouldIntercept: func(cmd *cobra.Command) bool {
			return isPersistentFlagChanged(cmd, cfg.flagName)
		},
		Intercept: func(cmd *cobra.Command, args []string) (bool, error) {
			return serveMCPIfRequested(cmd, cfg, cmd.InOrStdin(), cmd.OutOrStdout())
		},
	})
}

func serveMCPIfRequested(c *cobra.Command, cfg *mcpConfig, in io.Reader, out io.Writer) (bool, error) {
	if !isPersistentFlagChanged(c, cfg.flagName) {
		return false, nil
	}
	return true, runMCPServer(c.Root(), cfg, in, out)
}

func isPersistentFlagChanged(c *cobra.Command, flagName string) bool {
	flagSets := []*pflag.FlagSet{
		c.Flags(),
		c.InheritedFlags(),
		c.Root().PersistentFlags(),
	}
	for _, fs := range flagSets {
		if fs == nil {
			continue
		}
		if flag := fs.Lookup(flagName); flag != nil && flag.Changed {
			return true
		}
	}
	return false
}

func runMCPServer(root *cobra.Command, cfg *mcpConfig, in io.Reader, out io.Writer) error {
	registry, err := newMCPRegistry(root, cfg)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(in)
	dec.UseNumber()
	enc := json.NewEncoder(out)

	for {
		var req structclimcp.Request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		resp, err := handleMCPRequest(root, cfg, registry, &req)
		if err != nil {
			return err
		}
		if resp == nil {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

func handleMCPRequest(root *cobra.Command, cfg *mcpConfig, registry *mcpRegistry, req *structclimcp.Request) (*structclimcp.Response, error) {
	if req == nil {
		return nil, nil
	}
	if req.JSONRPC != "" && req.JSONRPC != jsonrpcVersion {
		return jsonRPCError(req.ID, rpcCodeInvalidRequest, "jsonrpc must be 2.0"), nil
	}

	switch req.Method {
	case "initialize":
		return &structclimcp.Response{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result: structclimcp.InitializeResult{
				ProtocolVersion: structclimcp.ProtocolVersion,
				ServerInfo: structclimcp.ServerInfo{
					Name:    cfg.name,
					Version: cfg.version,
				},
				Capabilities: map[string]any{
					"tools": map[string]any{},
				},
			},
		}, nil
	case "notifications/initialized":
		return nil, nil
	case "tools/list":
		return &structclimcp.Response{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result:  structclimcp.ToolsListResult{Tools: registry.tools},
		}, nil
	case "tools/call":
		var params structclimcp.ToolCallParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return jsonRPCError(req.ID, rpcCodeInvalidParams, "invalid tools/call params"), nil
			}
		}
		result, rpcErr := callMCPTool(root, cfg, registry, params)
		if rpcErr != nil {
			return jsonRPCError(req.ID, rpcErr.Code, rpcErr.Message), nil
		}
		return &structclimcp.Response{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result:  result,
		}, nil
	default:
		if len(req.ID) == 0 {
			return nil, nil
		}
		return jsonRPCError(req.ID, rpcCodeMethodNotFound, "method not found"), nil
	}
}

func jsonRPCError(id json.RawMessage, code int, message string) *structclimcp.Response {
	return &structclimcp.Response{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error: &structclimcp.ResponseError{
			Code:    code,
			Message: message,
		},
	}
}

func newMCPRegistry(root *cobra.Command, cfg *mcpConfig) (*mcpRegistry, error) {
	schemas, err := JSONSchema(root, jsonschema.WithFullTree())
	if err != nil {
		return nil, fmt.Errorf("building MCP tool schemas: %w", err)
	}

	cmds := buildMCPCommandMap(root)
	registry := &mcpRegistry{
		defs: make(map[string]*mcpToolDef),
	}

	for _, schema := range schemas {
		cmd := cmds[schema.CommandPath]
		if !shouldIncludeMCPCommand(schema, cmd, cfg) {
			continue
		}

		name := mcpToolName(schema.CommandPath, root.Name(), cfg.separator)
		if _, excluded := cfg.exclude[name]; excluded {
			continue
		}
		if _, excluded := cfg.exclude[schema.CommandPath]; excluded {
			continue
		}

		inputSchema, err := schema.ToJSONSchema()
		if err != nil {
			return nil, fmt.Errorf("building MCP input schema for %s: %w", schema.CommandPath, err)
		}

		registry.tools = append(registry.tools, structclimcp.Tool{
			Name:        name,
			Description: schema.Description,
			InputSchema: json.RawMessage(inputSchema),
		})
		registry.defs[name] = &mcpToolDef{
			name:   name,
			schema: schema,
			path:   mcpCommandPathArgs(schema.CommandPath),
		}
	}

	return registry, nil
}

func buildMCPCommandMap(root *cobra.Command) map[string]*cobra.Command {
	m := make(map[string]*cobra.Command)
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		m[c.CommandPath()] = c
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return m
}

func shouldIncludeMCPCommand(schema *CommandSchema, cmd *cobra.Command, cfg *mcpConfig) bool {
	if schema == nil || cmd == nil {
		return false
	}
	if cmd.Hidden || cmd.Name() == "help" || cmd.IsAdditionalHelpTopicCommand() {
		return false
	}
	if !cmd.Runnable() {
		return false
	}
	if cfg.allCommands {
		return true
	}
	return len(schema.Subcommands) == 0
}

func mcpToolName(commandPath, rootName, separator string) string {
	parts := strings.Fields(commandPath)
	if len(parts) <= 1 {
		return rootName
	}
	return strings.Join(parts[1:], separator)
}

func mcpCommandPathArgs(commandPath string) []string {
	parts := strings.Fields(commandPath)
	if len(parts) <= 1 {
		return nil
	}
	return append([]string(nil), parts[1:]...)
}

func callMCPTool(root *cobra.Command, cfg *mcpConfig, registry *mcpRegistry, params structclimcp.ToolCallParams) (*structclimcp.ToolCallResult, *structclimcp.ResponseError) {
	if params.Name == "" {
		return nil, &structclimcp.ResponseError{Code: rpcCodeInvalidParams, Message: "tool name is required"}
	}
	def := registry.defs[params.Name]
	if def == nil {
		return nil, &structclimcp.ResponseError{Code: rpcCodeInvalidParams, Message: "unknown tool"}
	}

	flagArgs, err := mcpArgumentsToArgs(def.schema, params.Arguments)
	if err != nil {
		return nil, &structclimcp.ResponseError{Code: rpcCodeInvalidParams, Message: err.Error()}
	}

	argv := append([]string(nil), def.path...)
	argv = append(argv, flagArgs...)

	stdout, stderr, executedCmd, execErr := executeMCPCommand(root, cfg, argv)
	if execErr != nil {
		var structured bytes.Buffer
		HandleError(executedCmd, execErr, &structured)
		return &structclimcp.ToolCallResult{
			Content: []structclimcp.ToolCallContent{{
				Type: "text",
				Text: strings.TrimSpace(structured.String()),
			}},
			IsError: true,
		}, nil
	}

	text := stdout.String()
	if stderr.Len() > 0 {
		text += stderr.String()
	}

	return &structclimcp.ToolCallResult{
		Content: []structclimcp.ToolCallContent{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

func executeMCPCommand(root *cobra.Command, cfg *mcpConfig, argv []string) (*bytes.Buffer, *bytes.Buffer, *cobra.Command, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if cfg != nil && cfg.commandFactory != nil {
		argvCopy := append([]string(nil), argv...)
		cmd, err := cfg.commandFactory(argvCopy, &stdout, &stderr)
		if err != nil {
			return &stdout, &stderr, root, err
		}
		if cmd == nil {
			return &stdout, &stderr, root, fmt.Errorf("command factory returned nil command")
		}
		cmd.SetArgs(argvCopy)
		cmd.SetIn(strings.NewReader(""))
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		executedCmd, err := cmd.ExecuteC()
		if executedCmd == nil {
			executedCmd = cmd
		}
		return &stdout, &stderr, executedCmd, err
	}

	if err := resetCommandExecutionState(root); err != nil {
		return nil, nil, root, err
	}

	root.SetArgs(append([]string(nil), argv...))
	root.SetIn(strings.NewReader(""))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SilenceErrors = true
	root.SilenceUsage = true

	cmd, err := root.ExecuteC()
	if err != nil {
		if cmd == nil {
			cmd = root
		}
		return &stdout, &stderr, cmd, err
	}

	return &stdout, &stderr, cmd, nil
}

func resetCommandExecutionState(root *cobra.Command) error {
	var walk func(*cobra.Command) error
	walk = func(c *cobra.Command) error {
		for _, fs := range []*pflag.FlagSet{c.LocalFlags(), c.PersistentFlags()} {
			if fs == nil {
				continue
			}
			var resetErr error
			fs.VisitAll(func(f *pflag.Flag) {
				if resetErr != nil {
					return
				}
				if err := f.Value.Set(f.DefValue); err != nil {
					resetErr = fmt.Errorf("resetting flag %s: %w", f.Name, err)
					return
				}
				f.Changed = false
			})
			if resetErr != nil {
				return resetErr
			}
		}
		for _, sub := range c.Commands() {
			if err := walk(sub); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

func mcpArgumentsToArgs(schema *CommandSchema, arguments map[string]any) ([]string, error) {
	if len(arguments) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(arguments))
	for key := range arguments {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var args []string
	for _, key := range keys {
		flagSchema := schema.Flags[key]
		if flagSchema == nil {
			return nil, fmt.Errorf("unknown argument %q", key)
		}

		value := arguments[key]
		if value == nil {
			continue
		}

		values, err := mcpArgumentValues(value)
		if err != nil {
			return nil, fmt.Errorf("invalid argument %q: %w", key, err)
		}
		for _, v := range values {
			args = append(args, "--"+key, v)
		}
	}

	return args, nil
}

func mcpArgumentValues(value any) ([]string, error) {
	switch v := value.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, err := mcpArgumentString(item)
			if err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, nil
	default:
		s, err := mcpArgumentString(value)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}
}

func mcpArgumentString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case json.Number:
		return v.String(), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case int:
		return strconv.Itoa(v), nil
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("unsupported value type %T", value)
		}
		return string(b), nil
	}
}
