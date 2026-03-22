package main

import (
	"fmt"
	"log"
	"time"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/config"
	"github.com/spf13/cobra"
)

type Options struct {
	Retries   []uint            `flag:"retries" flagenv:"true"`
	Backoffs  []time.Duration   `flag:"backoffs" flagenv:"true"`
	FeatureOn []bool            `flag:"feature-on" flagenv:"true"`
	Labels    map[string]string `flag:"labels" flagenv:"true"`
	Limits    map[string]int    `flag:"limits" flagenv:"true"`
	Counts    map[string]int64  `flag:"counts" flagenv:"true"`
}

func (o *Options) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func NewRootCmd() (*cobra.Command, error) {
	opts := &Options{}

	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "Demonstrate slice and map contracts across flags, env, and config",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			_, message, err := structcli.UseConfigSimple(c)
			if err != nil {
				return err
			}
			if message != "" {
				fmt.Fprintln(c.OutOrStdout(), message)
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := structcli.Unmarshal(c, opts); err != nil {
				return err
			}

			fmt.Fprintf(c.OutOrStdout(), "retries=%v\n", opts.Retries)
			fmt.Fprintf(c.OutOrStdout(), "backoffs=%v\n", opts.Backoffs)
			fmt.Fprintf(c.OutOrStdout(), "feature_on=%v\n", opts.FeatureOn)
			fmt.Fprintf(c.OutOrStdout(), "labels=%v\n", opts.Labels)
			fmt.Fprintf(c.OutOrStdout(), "limits=%v\n", opts.Limits)
			fmt.Fprintf(c.OutOrStdout(), "counts=%v\n", opts.Counts)

			return nil
		},
	}

	structcli.SetupConfig(rootCmd, config.Options{AppName: "myapp"})
	if err := opts.Attach(rootCmd); err != nil {
		return nil, err
	}

	return rootCmd, nil
}

func main() {
	log.SetFlags(0)
	rootCmd, err := NewRootCmd()
	if err != nil {
		log.Fatalln(err)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
