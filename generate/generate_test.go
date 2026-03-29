package generate_test

import (
	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// testServeOptions defines flags via structcli.Define() — the same path real users take.
// This ensures annotations (env vars, defaults, required) are set correctly.
type testServeOptions struct {
	Port int    `flagshort:"p" flagdescr:"Server port" flagenv:"true" flagrequired:"true" default:"3000"`
	Host string `flag:"host" flagdescr:"Server host" flagenv:"true" default:"localhost"`
}

func (o *testServeOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// testConfigOptions for a secondary subcommand.
type testConfigOptions struct {
	Format string `flag:"format" flagdescr:"Output format" default:"json"`
}

func (o *testConfigOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// buildTestTree creates a realistic CLI tree using structcli.Define().
// All annotations (env vars, defaults, required, paths) are set automatically.
func buildTestTree() *cobra.Command {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{
		Use:   "myapp",
		Short: "A test CLI application",
		RunE:  noop,
	}

	serve := &cobra.Command{
		Use:     "serve",
		Short:   "Start the server",
		Example: "myapp serve --port 8080 --host 0.0.0.0",
		RunE:    noop,
	}
	serveOpts := &testServeOptions{}
	serveOpts.Attach(serve)

	config := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		RunE:  noop,
	}
	configOpts := &testConfigOptions{}
	configOpts.Attach(config)

	root.AddCommand(serve, config)

	return root
}
