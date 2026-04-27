package main

import (
	"fmt"
	"log"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level `flag:"level" flagdescr:"Set logging level" flagenv:"true"`
	Port     int           `flagshort:"p" flagdescr:"Server port" flagenv:"true" default:"3000"`
}

func main() {
	log.SetFlags(0)
	opts := &Options{}
	cli := &cobra.Command{
		Use:   "myapp",
		Short: "A simple CLI example",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), opts)

			return nil
		},
	}

	if err := structcli.Setup(cli,
		structcli.WithJSONSchema(),
		structcli.WithMCP(),
		structcli.WithFlagErrors(),
	); err != nil {
		log.Fatalln(err)
	}

	structcli.Bind(cli, opts)

	// Structured errors: JSON to stderr + semantic exit code on failure
	structcli.ExecuteOrExit(cli)
}
