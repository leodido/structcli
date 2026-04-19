package structcli

import "go.uber.org/zap/zapcore"

// Built-in enum registrations for well-known types.
//
// zapcore.Level uses a hardcoded map rather than delegating to
// zapcore.ParseLevel. The level set has been stable since zapcore v1.
// If a future version adds levels, this map must be updated.
func init() {
	RegisterIntEnum[zapcore.Level](map[zapcore.Level][]string{
		zapcore.DebugLevel:  {"debug"},
		zapcore.InfoLevel:   {"info"},
		zapcore.WarnLevel:   {"warn"},
		zapcore.ErrorLevel:  {"error"},
		zapcore.DPanicLevel: {"dpanic"},
		zapcore.PanicLevel:  {"panic"},
		zapcore.FatalLevel:  {"fatal"},
	})
}
