package full_example_cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/structcli"
	"github.com/leodido/structcli/config"
	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/flagkit"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "prod"
)

func init() {
	structcli.RegisterEnum[Environment](map[Environment][]string{
		EnvDevelopment: {"dev", "development"},
		EnvStaging:    {"staging", "stage"},
		EnvProduction: {"prod", "production"},
	})
}

type EvenDeeper struct {
	Setting   string `flag:"deeper-setting" default:"default-deeper-setting"`
	NoDefault string
}

type Deeply struct {
	Setting string `flag:"deep-setting" default:"default-deep-setting"`
	Deeper  EvenDeeper
}

type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Env-only field: settable only via environment variable, not CLI flag
	SecretKey string `flagenv:"only" flag:"secret-key" flagdescr:"Secret signing key (env only)"`

	// Same in-memory type (bytes), different textual contracts at the CLI boundary.
	TokenHex    structcli.Hex    `flag:"token-hex" flaggroup:"Security" flagdescr:"Token bytes encoded as hex" flagenv:"true" default:"68656c6c6f"`
	TokenBase64 structcli.Base64 `flag:"token-base64" flaggroup:"Security" flagdescr:"Token bytes encoded as base64" flagenv:"true" default:"aGVsbG8="`

	// Network contracts using net families.
	BindIP        net.IP     `flag:"bind-ip" flaggroup:"Network" flagdescr:"Bind interface IP" flagenv:"true" default:"127.0.0.1"`
	BindMask      net.IPMask `flag:"bind-mask" flaggroup:"Network" flagdescr:"Bind interface mask" flagenv:"true" default:"ffffff00"`
	AdvertiseCIDR net.IPNet  `flag:"advertise-cidr" flaggroup:"Network" flagdescr:"Advertised service subnet (CIDR)" flagenv:"true" default:"127.0.0.0/24"`
	TrustedPeers  []net.IP   `flag:"trusted-peers" flaggroup:"Network" flagdescr:"Trusted peer IPs (comma separated)" flagenv:"true" default:"127.0.0.2,127.0.0.3"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path" flagenv:"true"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`

	// Custom type
	TargetEnv Environment `flag:"target-env" flagdescr:"Set the target environment" default:"dev"`

	Deep Deeply
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flagdescr:"Max database connections" default:"10" flagenv:"true"`
}



// Attach makes ServerOptions implement the Options interface
func (o *ServerOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func makeSrvC() *cobra.Command {
	commonOpts := &UtilityFlags{}
	opts := &ServerOptions{}

	srvC := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		Long:  "Start the server with the specified configuration",
		PreRunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|--srvC.PreRunE")
			if err := structcli.Unmarshal(c, opts); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))
			fmt.Fprintf(c.OutOrStdout(), "Decoded tokens: hex=%q base64=%q\n", string(opts.TokenHex), string(opts.TokenBase64))
			fmt.Fprintf(c.OutOrStdout(), "Decoded network: ip=%s mask=%s cidr=%s peers=%s\n",
				opts.BindIP.String(),
				net.IPMask(opts.BindMask).String(),
				opts.AdvertiseCIDR.String(),
				joinIPs(opts.TrustedPeers))

			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			fmt.Fprintln(c.OutOrStdout(), "|--srvC.RunE")
		},
	}
	opts.Attach(srvC)

	versionC := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|---versionC.RunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(commonOpts))

			return nil
		},
	}

	commonOpts.Attach(versionC)
	srvC.AddCommand(versionC)

	return srvC
}

var _ structcli.ValidatableOptions = (*UserConfig)(nil)
var _ structcli.TransformableOptions = (*UserConfig)(nil)

type UserConfig struct {
	Email string `flag:"email" flagdescr:"User email" validate:"email"`
	Age   int    `flag:"age" flagdescr:"User age" validate:"min=18,max=120"`
	Name  string `flag:"name" flagdescr:"User name" mod:"trim,title"`
}

// Transform makes UserConfig implement the ValidatableOptions interface
//
// The UserConfig options (flags/envs/configs) will be validated at unmarshalling time.
func (o *UserConfig) Validate(ctx context.Context) []error {
	var errs []error
	err := validator.New().Struct(o)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, fieldErr)
			}
		} else {
			errs = append(errs, fmt.Errorf("validator.Struct() failed unexpectedly: %w", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return errs
}

// Transform makes UserConfig implement the TransformableOptions interface
//
// The UserConfig options (flags/envs/configs) will be at molded at unmarshalling time (before validation).
func (o *UserConfig) Transform(ctx context.Context) error {
	return modifiers.New().Struct(ctx, o)
}

// Attach makes UserConfig implement the Options interface
func (o *UserConfig) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func makeUsrC() *cobra.Command {
	// Options implementing CommonOptions propagate automatically via commands context
	commonOpts := &UtilityFlags{}
	opts := &UserConfig{}

	usrC := &cobra.Command{
		Use:   "usr",
		Short: "User management",
		Long:  "Commands for managing users in the server",
	}

	addC := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		Long:  "Add a new user to the system with the specified details",
		PreRunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|---add.PreRunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(commonOpts))

			return structcli.Unmarshal(c, opts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|---add.RunE")
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))

			return nil
		},
	}

	opts.Attach(addC)
	commonOpts.Attach(addC)

	usrC.AddCommand(addC)
	// Setup of the usage text happens at structcli.Define
	// For the `usr` command we do it explicitly since it has no local flags
	structcli.SetupUsage(usrC)

	return usrC
}

var _ structcli.ValidatableOptions = (*PresetDemoOptions)(nil)
var _ structcli.TransformableOptions = (*PresetDemoOptions)(nil)

// PresetDemoOptions demonstrates flagpreset values flowing through
// transform and validation logic.
type PresetDemoOptions struct {
	Role  string `flag:"role" flagpreset:"as-admin=admin;as-guest=guest;as-super=superadmin" validate:"required,oneof=admin guest"`
	Label string `flag:"label" flagpreset:"auto-label=  john doe  " mod:"trim,title" validate:"required,min=3,max=32"`
}

func (o *PresetDemoOptions) Validate(ctx context.Context) []error {
	var errs []error
	err := validator.New().Struct(o)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, fieldErr)
			}
		} else {
			errs = append(errs, fmt.Errorf("validator.Struct() failed unexpectedly: %w", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (o *PresetDemoOptions) Transform(ctx context.Context) error {
	return modifiers.New().Struct(ctx, o)
}

func (o *PresetDemoOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func makePresetC() *cobra.Command {
	opts := &PresetDemoOptions{}

	presetC := &cobra.Command{
		Use:   "preset",
		Short: "Demonstrate flag presets with validation and transformation",
		Long:  "Demonstrate that flagpreset aliases are syntactic sugar and still flow through Transform and Validate",
		RunE: func(c *cobra.Command, args []string) error {
			if err := structcli.Unmarshal(c, opts); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))

			return nil
		},
	}
	opts.Attach(presetC)

	return presetC
}

// LogsOptions demonstrates flagkit.Follow composition.
type LogsOptions struct {
	flagkit.Follow
	Service string `flag:"service" flagshort:"s" flagdescr:"Service name to show logs for" flagrequired:"true"`
}

func (o *LogsOptions) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}
	flagkit.AnnotateCommand(c)

	return nil
}

func makeLogsC() *cobra.Command {
	opts := &LogsOptions{}

	logsC := &cobra.Command{
		Use:   "logs",
		Short: "Show service logs",
		Long:  "Display logs for a service, optionally streaming with --follow",
		Example: `  full logs --service api
  full logs -s api --follow
  full logs -s api -f`,
		RunE: func(c *cobra.Command, args []string) error {
			if err := structcli.Unmarshal(c, opts); err != nil {
				return err
			}
			if opts.Follow.Enabled {
				fmt.Fprintf(c.OutOrStdout(), "Streaming logs for service %q...\n", opts.Service)
			} else {
				fmt.Fprintf(c.OutOrStdout(), "Showing recent logs for service %q\n", opts.Service)
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))

			return nil
		},
	}
	opts.Attach(logsC)

	return logsC
}

var _ structcli.ContextOptions = (*UtilityFlags)(nil)

type UtilityFlags struct {
	Verbose int  `flagtype:"count" flagshort:"v" flaggroup:"Utility"`
	DryRun  bool `flag:"dry" flaggroup:"Utility" flagenv:"true"`
}

type utilityFlagsKey struct{}

func (f *UtilityFlags) Attach(c *cobra.Command) error {
	return structcli.Define(c, f)
}

// Context implements the CommonOptions interface
func (f *UtilityFlags) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, utilityFlagsKey{}, f)
}

func (f *UtilityFlags) FromContext(ctx context.Context) error {
	value, ok := ctx.Value(utilityFlagsKey{}).(*UtilityFlags)
	if !ok {
		return fmt.Errorf("couldn't obtain from context")
	}
	*f = *value

	return nil
}

func NewRootC(exitOnDebug bool) (*cobra.Command, error) {
	// Options implementing CommonOptions propagate automatically via commands context
	commonOpts := &UtilityFlags{}

	rootC := &cobra.Command{
		Use:               "full",
		Short:             "A beautiful CLI application",
		Long:              "A demonstration of the structcli library with beautiful CLI features",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		// Parse its own flags first, then continue traversing down to find subcommands
		// Useful for allowing context options not being attached to all the subcommands
		// Eg, `go run main.go --dry-run usr add` would fail otherwise
		TraverseChildren: true,
		// Because we handle errors ourselves in this example
		SilenceErrors: true,
	}

	// Global persistent pre-run for config file support
	rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		fmt.Fprintln(c.OutOrStdout(), "|-rootC.PersistentPreRunE")

		// Load config file if found
		_, configMessage, configErr := structcli.UseConfigSimple(c)
		if configErr != nil {
			return configErr
		}
		if configMessage != "" {
			c.Println(configMessage)
		}

		if err := structcli.Unmarshal(c, commonOpts); err != nil {
			return err
		}

		return nil
	}
	rootC.RunE = func(c *cobra.Command, args []string) error {
		fmt.Fprintln(c.OutOrStdout(), "|-rootC.RunE")

		return nil
	}

	// Initialize config before defining options so env annotations include the app prefix.
	if err := structcli.SetupConfig(rootC, config.Options{AppName: "full"}); err != nil {
		return nil, err
	}

	commonOpts.Attach(rootC)
	rootC.AddCommand(makeSrvC())
	rootC.AddCommand(makeUsrC())
	rootC.AddCommand(makePresetC())
	rootC.AddCommand(makeLogsC())

	// This single line enables the debugging global flag
	if err := structcli.SetupDebug(rootC, debug.Options{Exit: exitOnDebug}); err != nil {
		return nil, err
	}

	// Enable --jsonschema for machine-readable self-description (AI-native CLIs)
	if err := structcli.SetupJSONSchema(rootC, jsonschema.Options{}); err != nil {
		return nil, err
	}

	return rootC, nil
}

func pretty(opts any) string {
	prettyOpts, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error marshalling options: %s", err.Error()))
	}

	return string(prettyOpts)
}

func joinIPs(ips []net.IP) string {
	if len(ips) == 0 {
		return ""
	}

	out := make([]string, len(ips))
	for i := range ips {
		out[i] = ips[i].String()
	}

	return strings.Join(out, ",")
}
