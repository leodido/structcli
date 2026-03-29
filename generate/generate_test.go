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

// buildMinimalTree creates a CLI with a single command, no flags, no subcommands.
func buildMinimalTree() *cobra.Command {
	return &cobra.Command{
		Use:   "bare",
		Short: "A bare CLI with no subcommands",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

// buildNoDescriptionTree creates a CLI where some commands have empty descriptions.
func buildNoDescriptionTree() *cobra.Command {
	root := &cobra.Command{
		Use:  "nodesc",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	sub := &cobra.Command{
		Use:  "sub",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	opts := &testServeOptions{}
	opts.Attach(sub)
	root.AddCommand(sub)

	return root
}

// buildDuplicateNameTree creates a CLI where two subcommands share the same name
// under different parents (eg. "db add" and "user add").
func buildDuplicateNameTree() *cobra.Command {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{Use: "mycli", Short: "CLI with duplicate names", RunE: noop}

	db := &cobra.Command{Use: "db", Short: "Database commands"}
	dbAdd := &cobra.Command{Use: "add", Short: "Add a database", RunE: noop}
	db.AddCommand(dbAdd)

	user := &cobra.Command{Use: "user", Short: "User commands"}
	userAdd := &cobra.Command{Use: "add", Short: "Add a user", RunE: noop}
	user.AddCommand(userAdd)

	root.AddCommand(db, user)

	return root
}
