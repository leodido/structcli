package structcli

import (
	"fmt"

	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/helptopics"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
)

// setupConfig holds the resolved configuration from SetupOption functions.
type setupConfig struct {
	debug      *debug.Options
	jsonSchema *jsonschema.Options
	mcp        *structclimcp.Options
	helpTopics *helptopics.Options
	flagErrors bool
}

// SetupOption configures a feature in Setup.
type SetupOption func(*setupConfig)

// WithDebug enables the debug flag (--debug-options) on the root command.
func WithDebug(opts debug.Options) SetupOption {
	return func(c *setupConfig) {
		c.debug = &opts
	}
}

// WithJSONSchema enables the --jsonschema flag on the root command.
// Pass jsonschema.Options{} for defaults.
func WithJSONSchema(opts ...jsonschema.Options) SetupOption {
	return func(c *setupConfig) {
		o := jsonschema.Options{}
		if len(opts) > 0 {
			o = opts[0]
		}
		c.jsonSchema = &o
	}
}

// WithMCP enables the --mcp flag on the root command.
// Pass mcp.Options{} for defaults.
func WithMCP(opts ...structclimcp.Options) SetupOption {
	return func(c *setupConfig) {
		o := structclimcp.Options{}
		if len(opts) > 0 {
			o = opts[0]
		}
		c.mcp = &o
	}
}

// WithHelpTopics enables help topic commands on the root command.
// Pass helptopics.Options{} for defaults.
func WithHelpTopics(opts ...helptopics.Options) SetupOption {
	return func(c *setupConfig) {
		o := helptopics.Options{}
		if len(opts) > 0 {
			o = opts[0]
		}
		c.helpTopics = &o
	}
}

// WithFlagErrors enables structured flag error interception on the root command.
func WithFlagErrors() SetupOption {
	return func(c *setupConfig) {
		c.flagErrors = true
	}
}

// Setup configures the root command with the selected features.
//
// It calls the underlying Setup* functions in the correct internal order.
// Individual Setup* functions remain available for power users.
//
// Ordering is handled internally:
//  1. Debug (registers --debug-options flag)
//  2. JSON Schema (registers --jsonschema flag, wraps execution)
//  3. Help Topics (adds help topic subcommands)
//  4. Flag Errors (intercepts flag parsing errors)
//  5. MCP (registers --mcp flag, wraps execution)
//
// WithConfig and WithAppName are not yet supported (PR 5).
func Setup(cmd *cobra.Command, opts ...SetupOption) error {
	cfg := &setupConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.debug != nil {
		if err := SetupDebug(cmd, *cfg.debug); err != nil {
			return fmt.Errorf("structcli.Setup: debug: %w", err)
		}
	}

	if cfg.jsonSchema != nil {
		if err := SetupJSONSchema(cmd, *cfg.jsonSchema); err != nil {
			return fmt.Errorf("structcli.Setup: jsonschema: %w", err)
		}
	}

	if cfg.helpTopics != nil {
		if err := SetupHelpTopics(cmd, *cfg.helpTopics); err != nil {
			return fmt.Errorf("structcli.Setup: helptopics: %w", err)
		}
	}

	if cfg.flagErrors {
		SetupFlagErrors(cmd)
	}

	if cfg.mcp != nil {
		if err := SetupMCP(cmd, *cfg.mcp); err != nil {
			return fmt.Errorf("structcli.Setup: mcp: %w", err)
		}
	}

	return nil
}
