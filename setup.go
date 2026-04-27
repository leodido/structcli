package structcli

import (
	"fmt"

	"github.com/leodido/structcli/config"
	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/helptopics"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
)

const configAutoLoadAnnotation = "structcli/config-auto-load"

// setupConfig holds the resolved configuration from SetupOption functions.
type setupConfig struct {
	appName    string
	config     *config.Options
	debug      *debug.Options
	jsonSchema *jsonschema.Options
	mcp        *structclimcp.Options
	helpTopics *helptopics.Options
	flagErrors bool
}

// SetupOption configures a feature in Setup.
type SetupOption func(*setupConfig)

// WithAppName sets the application name used for environment variable prefixes
// and config file discovery. When flags already exist on the command tree (from
// earlier Bind calls), their env annotations are retroactively patched to include
// the new prefix.
//
// If a sub-option (e.g., debug.Options.AppName or config.Options.AppName) specifies
// a different name, Setup returns an error.
func WithAppName(name string) SetupOption {
	return func(c *setupConfig) {
		c.appName = name
	}
}

// WithConfig enables config file discovery and the --config flag on the root command.
// The actual config loading (UseConfigSimple) is deferred to ExecuteC's bind pipeline,
// before the first auto-unmarshal.
func WithConfig(opts ...config.Options) SetupOption {
	return func(c *setupConfig) {
		o := config.Options{}
		if len(opts) > 0 {
			o = opts[0]
		}
		c.config = &o
	}
}

// WithDebug enables the debug flag (--debug-options) on the root command.
//
// Unlike other With* options, WithDebug requires an explicit debug.Options
// argument because AppName and Exit have no sensible zero-value defaults.
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
// Most With* options are variadic: call them with no arguments for defaults (e.g., WithJSONSchema())
// WithDebug is the exception: it requires an explicit debug.Options because AppName and Exit have no sensible defaults.
//
// Ordering is handled internally:
//  1. AppName + env annotation patching (if flags already exist from earlier Bind calls)
//  2. Config (registers --config flag, defers auto-load to ExecuteC)
//  3. Debug (registers --debug-options flag)
//  4. JSON Schema (registers --jsonschema flag, wraps execution)
//  5. Help Topics (adds help topic subcommands)
//  6. Flag Errors (intercepts flag parsing errors)
//  7. MCP (registers --mcp flag, wraps execution)
func Setup(cmd *cobra.Command, opts ...SetupOption) error {
	cfg := &setupConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate AppName conflicts with sub-options.
	if cfg.appName != "" {
		if err := validateAppNameConflicts(cfg); err != nil {
			return err
		}
	}

	// Apply AppName: set the global prefix and patch existing flags.
	if cfg.appName != "" {
		if err := applyAppName(cmd, cfg.appName); err != nil {
			return fmt.Errorf("structcli.Setup: appname: %w", err)
		}
	}

	// Propagate AppName into sub-options that accept it.
	if cfg.appName != "" {
		if cfg.config != nil && cfg.config.AppName == "" {
			cfg.config.AppName = cfg.appName
		}
		if cfg.debug != nil && cfg.debug.AppName == "" {
			cfg.debug.AppName = cfg.appName
		}
	}

	if cfg.config != nil {
		if err := SetupConfig(cmd, *cfg.config); err != nil {
			return fmt.Errorf("structcli.Setup: config: %w", err)
		}
		// Mark root for deferred auto-load in ExecuteC.
		if cmd.Annotations == nil {
			cmd.Annotations = make(map[string]string)
		}
		cmd.Annotations[configAutoLoadAnnotation] = "true"
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

// validateAppNameConflicts checks that sub-option AppName fields don't conflict
// with the top-level WithAppName value.
func validateAppNameConflicts(cfg *setupConfig) error {
	if cfg.debug != nil && cfg.debug.AppName != "" && cfg.debug.AppName != cfg.appName {
		return fmt.Errorf("structcli.Setup: WithAppName(%q) conflicts with debug.Options.AppName(%q)", cfg.appName, cfg.debug.AppName)
	}
	if cfg.config != nil && cfg.config.AppName != "" && cfg.config.AppName != cfg.appName {
		return fmt.Errorf("structcli.Setup: WithAppName(%q) conflicts with config.Options.AppName(%q)", cfg.appName, cfg.config.AppName)
	}

	return nil
}

// applyAppName sets the global env prefix and retroactively patches env
// annotations on any flags already defined on the command tree (from earlier
// Bind calls). For each patched command, it clears bound-env state and re-runs
// BindEnv so viper picks up the corrected env var names.
func applyAppName(cmd *cobra.Command, appName string) error {
	oldPrefix := internalenv.GetPrefix() // e.g. "" or "OLD_"
	SetEnvPrefix(appName)
	newPrefix := internalenv.GetPrefix() // e.g. "MYAPP_"

	if oldPrefix == newPrefix {
		return nil // no change needed
	}

	// Walk the entire command tree and patch env annotations.
	patchTreeEnvPrefix(cmd, oldPrefix, newPrefix)

	return nil
}

// patchTreeEnvPrefix recursively patches env annotations on cmd and all descendants.
func patchTreeEnvPrefix(c *cobra.Command, oldPrefix, newPrefix string) {
	internalenv.PatchEnvPrefix(c, oldPrefix, newPrefix)
	// Re-bind env vars with the updated annotations.
	// Errors here are non-fatal — flags without env annotations are simply skipped.
	_ = internalenv.BindEnv(c)

	for _, sub := range c.Commands() {
		patchTreeEnvPrefix(sub, oldPrefix, newPrefix)
	}
}
