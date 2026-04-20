package flagkit

import (
	"log/slog"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

func init() {
	registerFlag("log-level")
}

// ZapLogLevel provides a --log-level flag backed by [zapcore.Level].
//
// The default is info. zapcore.Level is registered as an integer enum
// by structcli's built-in init(), so no additional registration is needed.
//
// Usage:
//
//	type Options struct {
//	    flagkit.ZapLogLevel
//	    Host string `flag:"host" flagdescr:"Server host"`
//	}
type ZapLogLevel struct {
	LogLevel zapcore.Level `flag:"log-level" flagdescr:"Set log level" default:"info" flagenv:"true" flaggroup:"Logging"`
}

// Attach implements [structcli.Options].
func (o *ZapLogLevel) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("log-level"); f != nil {
		_ = c.Flags().SetAnnotation("log-level", FlagKitAnnotation, []string{"true"})
	}

	return nil
}

// LogLevel is the recommended log level type. It is an alias for [ZapLogLevel].
//
// Embed this in your options struct for the standard --log-level flag backed
// by zapcore.Level. Use [SlogLogLevel] if you prefer the stdlib slog package.
type LogLevel = ZapLogLevel

// SlogLogLevel provides a --log-level flag backed by [slog.Level] (stdlib).
//
// The default is info. slog.Level is handled by structcli's built-in hooks,
// so no additional registration is needed.
//
// Usage:
//
//	type Options struct {
//	    flagkit.SlogLogLevel
//	}
type SlogLogLevel struct {
	LogLevel slog.Level `flag:"log-level" flagdescr:"Set log level" default:"info" flagenv:"true" flaggroup:"Logging"`
}

// Attach implements [structcli.Options].
func (o *SlogLogLevel) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("log-level"); f != nil {
		_ = c.Flags().SetAnnotation("log-level", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
