package internaldebug

import (
	"testing"

	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFlagSource_Flag(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("port", "8080", "listen port")
	require.NoError(t, cmd.Flags().Set("port", "9090"))

	f := cmd.Flags().Lookup("port")
	assert.Equal(t, SourceFlag, ResolveFlagSource(f, nil))
}

func TestResolveFlagSource_Env(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("timeout", "30s", "timeout")
	_ = cmd.Flags().SetAnnotation("timeout", internalenv.FlagAnnotation, []string{"APP_TIMEOUT"})

	t.Setenv("APP_TIMEOUT", "60s")

	f := cmd.Flags().Lookup("timeout")
	assert.Equal(t, SourceEnv, ResolveFlagSource(f, nil))
}

func TestResolveFlagSource_Config(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("log-level", "info", "log level")

	configV := viper.New()
	configV.Set("log-level", "debug")

	f := cmd.Flags().Lookup("log-level")
	assert.Equal(t, SourceConfig, ResolveFlagSource(f, configV))
}

func TestResolveFlagSource_Default(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("verbose", "false", "verbose")

	f := cmd.Flags().Lookup("verbose")
	assert.Equal(t, SourceDefault, ResolveFlagSource(f, nil))
}

func TestResolveFlagSource_FlagOverridesEnv(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("port", "8080", "port")
	_ = cmd.Flags().SetAnnotation("port", internalenv.FlagAnnotation, []string{"APP_PORT"})
	require.NoError(t, cmd.Flags().Set("port", "9090"))

	t.Setenv("APP_PORT", "7070")

	f := cmd.Flags().Lookup("port")
	assert.Equal(t, SourceFlag, ResolveFlagSource(f, nil))
}

func TestResolveFlagSource_EnvOverridesConfig(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("port", "8080", "port")
	_ = cmd.Flags().SetAnnotation("port", internalenv.FlagAnnotation, []string{"APP_PORT"})

	t.Setenv("APP_PORT", "7070")

	configV := viper.New()
	configV.Set("port", "6060")

	f := cmd.Flags().Lookup("port")
	assert.Equal(t, SourceEnv, ResolveFlagSource(f, configV))
}
