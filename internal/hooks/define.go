package internalhooks

import (
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"slices"
	"strings"
	"time"

	structclivalues "github.com/leodido/structcli/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag/v2"
)

// DefineHookFunc defines how to create a flag for a custom type.
//
// It receives flag metadata and struct field information and must return a pflag.Value
// that knows how to set the underlying field's value, along with an optional enhanced
// description for the flag's usage message.
//
// The short flag name is not passed here.
// The caller registers the returned pflag.Value with the appropriate short name via VarP.
type DefineHookFunc func(name, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string)

// DefineHookRegistry maps types to their flag definition functions.
var DefineHookRegistry = map[reflect.Type]DefineHookFunc{
	reflect.TypeFor[time.Duration]():     DefineTimeDurationHookFunc(),
	reflect.TypeFor[[]time.Duration]():   DefineDurationSliceHookFunc(),
	reflect.TypeFor[[]bool]():            DefineBoolSliceHookFunc(),
	reflect.TypeFor[[]uint]():            DefineUintSliceHookFunc(),
	reflect.TypeFor[map[string]string](): DefineStringMapHookFunc(),
	reflect.TypeFor[map[string]int]():    DefineIntMapHookFunc(),
	reflect.TypeFor[map[string]int64]():  DefineInt64MapHookFunc(),
	reflect.TypeFor[net.IP]():            DefineIPHookFunc(),
	reflect.TypeFor[net.IPMask]():        DefineIPMaskHookFunc(),
	reflect.TypeFor[net.IPNet]():         DefineIPNetHookFunc(),
	reflect.TypeFor[[]net.IP]():          DefineIPSliceHookFunc(),
	reflect.TypeFor[slog.Level]():        DefineSlogLevelHookFunc(),
	reflect.TypeFor[[]uint8]():           DefineRawBytesHookFunc(),
}

// defineHookRegistryByName is a string-keyed fallback for types whose
// reflect.Type is not available in this package (circular import).
var defineHookRegistryByName = map[string]DefineHookFunc{}

var byteSliceType = reflect.TypeOf([]byte(nil))

func defineByteSliceValueHookFunc(newValue func(val []byte, ref *[]byte) pflag.Value) DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Convert(byteSliceType).Interface().([]byte)
		// The field may be a named type (e.g. structcli.Hex) so Addr().Interface()
		// would yield *Hex, not *[]byte. Use reflect.NewAt to reinterpret the
		// pointer as *[]byte, which is safe because the underlying type is []byte.
		ref := reflect.NewAt(byteSliceType, fieldValue.Addr().UnsafePointer()).Interface().(*[]byte)

		return newValue(val, ref), descr
	}
}

func defineFlagSetValueHookFunc[T any](register func(fs *pflag.FlagSet, ref *T, name, short string, val T, usage string)) DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(T)
		ref := fieldValue.Addr().Interface().(*T)
		fs := pflag.NewFlagSet(name, pflag.ContinueOnError)
		register(fs, ref, name, "", val, descr)

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
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IP)
		ref := fieldValue.Addr().Interface().(*net.IP)

		return structclivalues.NewIP(val, ref), descr
	}
}

func DefineIPMaskHookFunc() DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IPMask)
		ref := fieldValue.Addr().Interface().(*net.IPMask)

		return structclivalues.NewIPMask(val, ref), descr
	}
}

func DefineIPNetHookFunc() DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(net.IPNet)
		ref := fieldValue.Addr().Interface().(*net.IPNet)

		return structclivalues.NewIPNet(val, ref), descr
	}
}

func DefineIPSliceHookFunc() DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().([]net.IP)
		ref := fieldValue.Addr().Interface().(*[]net.IP)

		return structclivalues.NewIPSlice(val, ref), descr
	}
}

func DefineTimeDurationHookFunc() DefineHookFunc {
	return func(name, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(time.Duration)
		ref := fieldValue.Addr().Interface().(*time.Duration)

		return structclivalues.NewDuration(val, ref), descr
	}
}

// enumHelpText builds a sorted "{val1,val2,...}" addendum from an enum map
// keyed by integer-typed levels. It returns the sorted value names and the
// description with the addendum appended.
func enumHelpText[L ~int | ~int8 | ~int16 | ~int32 | ~int64](levels map[L][]string, descr string) ([]string, string) {
	keys := make([]int64, 0, len(levels))
	for k := range levels {
		keys = append(keys, int64(k))
	}
	slices.Sort(keys)

	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, levels[L(k)][0])
	}

	return values, descr + fmt.Sprintf(" {%s}", strings.Join(values, ","))
}

// DefineSlogLevelHookFunc creates a flag definition function for slog.Level.
//
// It returns an enum flag that implements pflag.Value.
func DefineSlogLevelHookFunc() DefineHookFunc {
	return func(name, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		logLevels := map[slog.Level][]string{
			slog.LevelDebug: {"debug"},
			slog.LevelInfo:  {"info"},
			slog.LevelWarn:  {"warn"},
			slog.LevelError: {"error"},
		}

		values, enhancedDescr := enumHelpText(logLevels, descr)

		fieldPtr := fieldValue.Addr().Interface().(*slog.Level)
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), logLevels, enumflag.EnumCaseInsensitive)

		return WrapWithEnumValues(enumFlag, values), enhancedDescr
	}
}

// DefineIntEnumHookFunc creates a DefineHookFunc for a registered integer-based enum.
// It wraps enumflag/v2 and attaches EnumValuer metadata via WrapWithEnumValues.
func DefineIntEnumHookFunc[E ~int | ~int8 | ~int16 | ~int32 | ~int64](values map[E][]string) DefineHookFunc {
	return func(name, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		fieldPtr := fieldValue.Addr().Interface().(*E)
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), values, enumflag.EnumCaseInsensitive)
		enumValues, enhancedDescr := enumHelpText(values, descr)

		return WrapWithEnumValues(enumFlag, enumValues), enhancedDescr
	}
}

// DefineStringEnumHookFunc creates a DefineHookFunc for a registered ~string enum.
// The returned hook creates an enumStringValue that validates on Set() and
// appends "{val1,val2,...}" to the flag description.
func DefineStringEnumHookFunc[E ~string](values map[E][]string) DefineHookFunc {
	return func(name, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		target := fieldValue.Addr().Interface().(*E)
		ev := structclivalues.NewEnumString(target, values)
		enhancedDescr := descr + fmt.Sprintf(" {%s}", strings.Join(ev.EnumValues(), ","))

		return ev, enhancedDescr
	}
}

// InferDefineHooks looks up a define hook for the field's type and registers
// the flag if found. Falls back to the string-keyed registry for types whose
// reflect.Type is unavailable in this package.
func InferDefineHooks(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) bool {
	if defineFunc, ok := DefineHookRegistry[structField.Type]; ok {
		value, usage := defineFunc(name, descr, structField, fieldValue)
		c.Flags().VarP(value, name, short, usage)

		return true
	}

	if defineFunc, ok := defineHookRegistryByName[structField.Type.String()]; ok {
		value, usage := defineFunc(name, descr, structField, fieldValue)
		c.Flags().VarP(value, name, short, usage)

		return true
	}

	return false
}
