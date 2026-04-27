package main

import (
	"fmt"
	"log"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level
	Port     int
}

func main() {
	opts := &Options{}
	cli := &cobra.Command{
		Use: "myapp",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println(opts) // already populated

			return nil
		},
	}

	if err := structcli.Bind(cli, opts); err != nil {
		log.Fatalln(err)
	}
	structcli.ExecuteOrExit(cli)
}
