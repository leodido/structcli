package flagkit_test

import (
	"log/slog"
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// --- ZapLogLevel tests ---

func TestZapLogLevel_DefaultInfo(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, zapcore.InfoLevel, opts.LogLevel)
}

func TestZapLogLevel_SetDebug(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--log-level", "debug"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, zapcore.DebugLevel, opts.LogLevel)
}

func TestZapLogLevel_SetError(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--log-level", "error"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, zapcore.ErrorLevel, opts.LogLevel)
}

func TestZapLogLevel_Standalone(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("log-level")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "", f.Shorthand)
	assert.Equal(t, "info", f.DefValue)
	assert.Contains(t, f.Usage, "Set log level")
}

func TestZapLogLevel_Annotation(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("log-level")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestZapLogLevel_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestZapLogLevel_Embedded(t *testing.T) {
	type serverOpts struct {
		flagkit.ZapLogLevel
		Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	}
	opts := &serverOpts{}
	cmd := &cobra.Command{Use: "srv"}
	require.NoError(t, structcli.Define(cmd, opts))
	flagkit.AnnotateCommand(cmd)

	require.NoError(t, cmd.Flags().Parse([]string{"--log-level", "warn", "--host", "0.0.0.0"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, zapcore.WarnLevel, opts.LogLevel)
	assert.Equal(t, "0.0.0.0", opts.Host)
}

func TestZapLogLevel_JSONSchema(t *testing.T) {
	opts := &flagkit.ZapLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["log-level"]
	assert.True(t, ok, "JSON schema should include the log-level flag")
}

// --- SlogLogLevel tests ---

func TestSlogLogLevel_DefaultInfo(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, slog.LevelInfo, opts.LogLevel)
}

func TestSlogLogLevel_SetDebug(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--log-level", "debug"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, slog.LevelDebug, opts.LogLevel)
}

func TestSlogLogLevel_SetWarn(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--log-level", "warn"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, slog.LevelWarn, opts.LogLevel)
}

func TestSlogLogLevel_Standalone(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("log-level")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "", f.Shorthand)
	assert.Equal(t, "info", f.DefValue)
	assert.Contains(t, f.Usage, "Set log level")
}

func TestSlogLogLevel_Annotation(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("log-level")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestSlogLogLevel_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.SlogLogLevel{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

// --- LogLevel alias test ---

func TestLogLevel_IsZapLogLevel(t *testing.T) {
	var ll flagkit.LogLevel
	var zll flagkit.ZapLogLevel

	// LogLevel is a type alias for ZapLogLevel; assignment must compile.
	ll = zll
	zll = ll
	_ = zll
}
