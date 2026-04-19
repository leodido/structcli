package structcli

import "go.uber.org/zap/zapcore"

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
