// This example demonstrates structcli's structured error output for AI agents.
//
// It shows:
//   - SetupFlagErrors: typed flag error interception (no regex at classification time)
//   - HandleError: classifies errors and writes JSON to stderr
//   - ExecuteOrExit: convenience wrapper for main()
//   - Semantic exit codes for agent decision trees
//
// Try these:
//
//	# Valid invocation
//	go run . srv --port 8080 --host localhost --level info
//
//	# Missing required flag (StructuredError exit_code 10)
//	go run . srv
//
//	# Invalid flag value — wrong type (StructuredError exit_code 11)
//	go run . srv --port abc
//
//	# Invalid flag value — via short flag (StructuredError exit_code 11)
//	go run . srv -p xyz
//
//	# Unknown flag (StructuredError exit_code 12)
//	go run . srv --nonexistent
//
//	# Validation failed — invalid email (StructuredError exit_code 13)
//	go run . usr add --email notanemail --age 25 --name "John"
//
//	# Validation failed — age out of range (StructuredError exit_code 13)
//	go run . usr add --email test@example.com --age 10 --name "John"
//
//	# Unknown command (StructuredError exit_code 14)
//	go run . nonexistent
//
//	# Invalid enum value (StructuredError exit_code 15)
//	go run . srv --port 8080 --level bogus
//
//	# Env var with invalid value (StructuredError exit_code 25)
//	MYAPP_SRV_PORT=abc go run . srv
//
//	# Multiple missing flags
//	go run . usr add
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/leodido/structcli"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

// --- Server options ---

type ServerOptions struct {
	Host     string        `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port     int           `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`
	LogLevel zapcore.Level `flag:"level" flagdescr:"Set log level" flagenv:"true"`
}

func (o *ServerOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// --- User options with validation ---

var _ structcli.ValidatableOptions = (*UserOptions)(nil)

type UserOptions struct {
	Email string `flag:"email" flagdescr:"User email" validate:"required,email"`
	Age   int    `flag:"age" flagdescr:"User age" validate:"required,min=18,max=120"`
	Name  string `flag:"name" flagdescr:"User name" validate:"required,min=2"`
}

func (o *UserOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func (o *UserOptions) Validate(ctx context.Context) []error {
	var errs []error
	err := validator.New().Struct(o)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, fieldErr)
			}
		} else {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return errs
}

func main() {
	log.SetFlags(0)

	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "Structured error demo",
		// No need for SilenceErrors/SilenceUsage — ExecuteOrExit sets them automatically
	}

	// AI-native features: JSON Schema + typed flag errors
	structcli.SetupJSONSchema(rootCmd, jsonschema.Options{})
	structcli.SetupFlagErrors(rootCmd)

	// --- srv subcommand ---
	srvOpts := &ServerOptions{}
	srvCmd := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		PreRunE: func(c *cobra.Command, args []string) error {
			return structcli.Unmarshal(c, srvOpts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(c.OutOrStdout(), "Server started on %s:%d (log level: %s)\n",
				srvOpts.Host, srvOpts.Port, srvOpts.LogLevel)

			return nil
		},
	}
	srvOpts.Attach(srvCmd)
	rootCmd.AddCommand(srvCmd)

	// --- usr add subcommand ---
	usrCmd := &cobra.Command{
		Use:   "usr",
		Short: "User management",
	}

	addOpts := &UserOptions{}
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		PreRunE: func(c *cobra.Command, args []string) error {
			return structcli.Unmarshal(c, addOpts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(c.OutOrStdout(), "User added: %s (%s, age %d)\n",
				addOpts.Name, addOpts.Email, addOpts.Age)

			return nil
		},
	}
	addOpts.Attach(addCmd)
	usrCmd.AddCommand(addCmd)
	rootCmd.AddCommand(usrCmd)

	// One line: execute, handle errors as JSON, exit with semantic code
	structcli.ExecuteOrExit(rootCmd)
}
