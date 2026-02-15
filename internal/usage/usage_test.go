package internalusage

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestHelpers(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("name", "", "name")
	usage := flagUsages(flags)
	assert.Contains(t, usage, "--name")
	assert.True(t, strings.HasSuffix(usage, "\n"))

	assert.Equal(t, "x   ", rpad("x", 4))

	var b bytes.Buffer
	require.NoError(t, tmpl(&b, "hello"))
	assert.Equal(t, "hello", b.String())
	assert.Error(t, tmpl(errWriter{}, "x"))
}

func TestSetupUsage(t *testing.T) {
	root := &cobra.Command{
		Use:     "app",
		Aliases: []string{"a"},
		Example: "app child",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	root.Flags().String("plain", "", "plain flag")
	root.Flags().String("grouped", "", "grouped flag")
	root.PersistentFlags().Bool("config", false, "config")
	require.NoError(t, root.Flags().SetAnnotation("grouped", FlagGroupAnnotation, []string{"Alpha"}))

	child := &cobra.Command{
		Use:   "child",
		Short: "child cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
	topic := &cobra.Command{
		Use:   "topic",
		Short: "topic help",
	}
	root.AddCommand(child, topic)

	Setup(root)
	usage := root.UsageString()

	assert.Contains(t, usage, "Usage:")
	assert.Contains(t, usage, "Aliases:")
	assert.Contains(t, usage, "Examples:")
	assert.Contains(t, usage, "Available Commands:")
	assert.Contains(t, usage, "Flags:")
	assert.Contains(t, usage, "Alpha Flags:")
	assert.Contains(t, usage, "Global Flags:")
	assert.Contains(t, usage, "Additional help topics:")
	assert.Contains(t, usage, "Use \"app [command] --help\"")
	assert.Contains(t, usage, "child")
	assert.Contains(t, usage, "topic")
}
