package internaldebug

import (
	"testing"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDebugActive_DefaultFlag(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)
	root.PersistentFlags().String("debug-options", "", "")

	require.NoError(t, root.PersistentFlags().Set("debug-options", "text"))
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_JSON(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)
	root.PersistentFlags().String("debug-options", "", "")

	require.NoError(t, root.PersistentFlags().Set("debug-options", "json"))
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_CustomFlagAnnotation(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{
		Use:         "run",
		Annotations: map[string]string{FlagAnnotation: "debug-mode"},
	}
	root.AddCommand(run)
	root.PersistentFlags().String("debug-mode", "", "")

	require.NoError(t, root.PersistentFlags().Set("debug-mode", "text"))
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_FromScopedViper(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)

	internalscope.Get(root).Viper().Set("debug-options", "true")
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_FalseWhenUnset(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)

	assert.False(t, IsDebugActive(run))
}

func TestGetFormat_Text(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().String("debug-options", "", "")
	require.NoError(t, root.PersistentFlags().Set("debug-options", "text"))
	assert.Equal(t, "text", GetFormat(root))
}

func TestGetFormat_JSON(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().String("debug-options", "", "")
	require.NoError(t, root.PersistentFlags().Set("debug-options", "json"))
	assert.Equal(t, "json", GetFormat(root))
}

func TestGetFormat_TruthyBackwardCompat(t *testing.T) {
	for _, val := range []string{"true", "1", "yes"} {
		t.Run(val, func(t *testing.T) {
			root := &cobra.Command{Use: "app"}
			root.PersistentFlags().String("debug-options", "", "")
			require.NoError(t, root.PersistentFlags().Set("debug-options", val))
			assert.Equal(t, "text", GetFormat(root))
		})
	}
}

func TestGetFormat_Empty(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().String("debug-options", "", "")
	assert.Equal(t, "", GetFormat(root))
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"json", "json"},
		{"JSON", "json"},
		{"text", "text"},
		{"TEXT", "text"},
		{"true", "text"},
		{"1", "text"},
		{"yes", "text"},
		{"", ""},
		{"false", ""},
		{"0", ""},
		{"no", ""},
		{"anything", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeFormat(tt.input))
		})
	}
}
