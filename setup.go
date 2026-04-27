package structcli

import (
	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/helptopics"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
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
