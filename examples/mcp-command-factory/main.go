package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/jsonschema"
	"github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
)

type Streams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type Options struct {
	Name string `flag:"name" flagdescr:"Name to greet" default:"agent"`
}

func (o *Options) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func NewRootCommand(streams Streams) *cobra.Command {
	opts := &Options{}
	root := &cobra.Command{
		Use:           "streamed",
		Short:         "Example CLI with streams captured during construction",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.ErrOut)

	greet := &cobra.Command{
		Use:   "greet",
		Short: "Write a greeting through captured streams",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return structcli.Unmarshal(cmd, opts)
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(streams.Out, "hello %s\n", opts.Name)
			fmt.Fprintln(streams.ErrOut, "diagnostic: greeting written")
		},
	}
	if err := opts.Attach(greet); err != nil {
		panic(err)
	}
	root.AddCommand(greet)
	return root
}

func main() {
	log.SetFlags(0)
	streams := Streams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	root := NewRootCommand(streams)

	if err := structcli.SetupJSONSchema(root, jsonschema.Options{}); err != nil {
		log.Fatalln(err)
	}
	if err := structcli.SetupMCP(root, mcp.Options{
		CommandFactory: func(argv []string, stdout io.Writer, stderr io.Writer) (*cobra.Command, error) {
			streams := Streams{In: strings.NewReader(""), Out: stdout, ErrOut: stderr}
			return NewRootCommand(streams), nil
		},
	}); err != nil {
		log.Fatalln(err)
	}

	structcli.ExecuteOrExit(root)
}
