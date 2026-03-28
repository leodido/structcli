package internalhooks

import (
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"
	"unsafe"

	structclivalues "github.com/leodido/structcli/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag/v2"
	"go.uber.org/zap/zapcore"
)

// FIXME: remove short from the signature?

// DefineHookFunc defines how to create a flag for a custom type.
//
// It receives flag metadata and struct field information and must return a pflag.Value
// that knows how to set the underlying field's value, along with an optional enhanced
// description for the flag's usage message.
type DefineHookFunc func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string)

// DefineHookRegistry keeps track of the built-in flag definition functions
var DefineHookRegistry = map[string]DefineHookFunc{
	"zapcore.Level":     DefineZapcoreLevelHookFunc(),
	"time.Duration":     DefineTimeDurationHookFunc(),
	"[]time.Duration":   DefineDurationSliceHookFunc(),
	"[]bool":            DefineBoolSliceHookFunc(),
	"[]uint":            DefineUintSliceHookFunc(),
	"map[string]string": DefineStringMapHookFunc(),
	"map[string]int":    DefineIntMapHookFunc(),
	"map[string]int64":  DefineInt64MapHookFunc(),
	"net.IP":            DefineIPHookFunc(),
	"net.IPMask":        DefineIPMaskHookFunc(),
	"net.IPNet":         DefineIPNetHookFunc(),
	"[]net.IP":          DefineIPSliceHookFunc(),
	"slog.Level":        DefineSlogLevelHookFunc(),
	"[]uint8":           DefineRawBytesHookFunc(),
	"structcli.Hex":     DefineHexBytesHookFunc(),
	"structcli.Base64":  DefineBase64BytesHookFunc(),
}

var byteSliceType = reflect.TypeOf([]byte(nil))

func defineByteSliceValueHookFunc(newValue func(val []byte, ref *[]byte) pflag.Value) DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Convert(byteSliceType).Interface().([]byte)
		ref := (*[]byte)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return newValue(val, ref), descr
	}
}

func defineFlagSetValueHookFunc[T any](register func(fs *pflag.FlagSet, ref *T, name, short string, val T, usage string)) DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(T)
		ref := (*T)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		fs := pflag.NewFlagSet(name, pflag.ContinueOnError)
		register(fs, ref, name, short, val, descr)

		return fs.Lookup(name).Value, descr
	}
}

func DefineRawBytesHookFunc() DefineHookFunc {
	return defineByteSliceValueHookFunc(func(val []byte, ref *[]byte) pflag.Value {
		return structclivalues.NewRawBytes(val, ref)
	})
}

func DefineDurationSliceHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *[]time.Duration, name, short string, val []time.Duration, usage string) {
		fs.DurationSliceVarP(ref, name, short, val, usage)
	})
}

func DefineBoolSliceHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *[]bool, name, short string, val []bool, usage string) {
		fs.BoolSliceVarP(ref, name, short, val, usage)
	})
}

func DefineUintSliceHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *[]uint, name, short string, val []uint, usage string) {
		fs.UintSliceVarP(ref, name, short, val, usage)
	})
}

func DefineStringMapHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *map[string]string, name, short string, val map[string]string, usage string) {
		fs.StringToStringVarP(ref, name, short, val, usage)
	})
}

func DefineIntMapHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *map[string]int, name, short string, val map[string]int, usage string) {
		fs.StringToIntVarP(ref, name, short, val, usage)
	})
}

func DefineInt64MapHookFunc() DefineHookFunc {
	return defineFlagSetValueHookFunc(func(fs *pflag.FlagSet, ref *map[string]int64, name, short string, val map[string]int64, usage string) {
		fs.StringToInt64VarP(ref, name, short, val, usage)
	})
}

func DefineHexBytesHookFunc() DefineHookFunc {
	return defineByteSliceValueHookFunc(func(val []byte, ref *[]byte) pflag.Value {
		return structclivalues.NewHexBytes(val, ref)
	})
}

func DefineBase64BytesHookFunc() DefineHookFunc {
	return defineByteSliceValueHookFunc(func(val []byte, ref *[]byte) pflag.Value {
		return structclivalues.NewBase64Bytes(val, ref)
	})
}

func DefineIPHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IP)
		ref := (*net.IP)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return structclivalues.NewIP(val, ref), descr
	}
}

func DefineIPMaskHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IPMask)
		ref := (*net.IPMask)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return structclivalues.NewIPMask(val, ref), descr
	}
}

func DefineIPNetHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IPNet)
		ref := (*net.IPNet)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return structclivalues.NewIPNet(val, ref), descr
	}
}

func DefineIPSliceHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().([]net.IP)
		ref := (*[]net.IP)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return structclivalues.NewIPSlice(val, ref), descr
	}
}

func DefineTimeDurationHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(time.Duration)
		ref := (*time.Duration)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return structclivalues.NewDuration(val, ref), descr
	}
}

// DefineZapcoreLevelHookFunc creates a flag definition function for zapcore.Level.
//
// It returns an enum flag that implements pflag.Value.
func DefineZapcoreLevelHookFunc() DefineHookFunc {
	return func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		logLevels := map[zapcore.Level][]string{
			zapcore.DebugLevel:  {"debug"},
			zapcore.InfoLevel:   {"info"},
			zapcore.WarnLevel:   {"warn"},
			zapcore.ErrorLevel:  {"error"},
			zapcore.DPanicLevel: {"dpanic"},
			zapcore.PanicLevel:  {"panic"},
			zapcore.FatalLevel:  {"fatal"},
		}

		keys := []int{}
		for k := range logLevels {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		values := []string{}
		for _, k := range keys {
			values = append(values, logLevels[zapcore.Level(k)][0])
		}
		addendum := fmt.Sprintf(" {%s}", strings.Join(values, ","))
		enhancedDescr := descr + addendum

		// Get pointer to the field for the enum flag
		fieldPtr := (*zapcore.Level)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), logLevels, enumflag.EnumCaseInsensitive)

		return WrapWithEnumValues(enumFlag, values), enhancedDescr
	}
}

// DefineSlogLevelHookFunc creates a flag definition function for slog.Level.
//
// It returns an enum flag that implements pflag.Value.
func DefineSlogLevelHookFunc() DefineHookFunc {
	return func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		logLevels := map[slog.Level][]string{
			slog.LevelDebug: {"debug"},
			slog.LevelInfo:  {"info"},
			slog.LevelWarn:  {"warn"},
			slog.LevelError: {"error"},
		}

		keys := []int{}
		for k := range logLevels {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		values := []string{}
		for _, k := range keys {
			values = append(values, logLevels[slog.Level(k)][0])
		}
		addendum := fmt.Sprintf(" {%s}", strings.Join(values, ","))
		enhancedDescr := descr + addendum

		// Get pointer to the field for the enum flag
		fieldPtr := (*slog.Level)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), logLevels, enumflag.EnumCaseInsensitive)

		return WrapWithEnumValues(enumFlag, values), enhancedDescr
	}
}

// InferDefineHooks checks if there's a predefined flag definition function for the given type
func InferDefineHooks(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) bool {
	if defineFunc, ok := DefineHookRegistry[structField.Type.String()]; ok {
		value, usage := defineFunc(name, short, descr, structField, fieldValue)
		c.Flags().VarP(value, name, short, usage)

		return true
	}

	return false
}
