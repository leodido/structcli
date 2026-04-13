package internalhooks

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

const (
	FlagDecodeHookAnnotation = "___leodido_structcli_flagdecodehooks"
)

type DecodeHookFunc func(input any) (any, error)

type decodingAnnotation struct {
	ann string
	fx  mapstructure.DecodeHookFunc
}

var DecodeHookRegistry = map[string]decodingAnnotation{
	"time.Duration": {
		"StringToTimeDurationHookFunc",
		mapstructure.StringToTimeDurationHookFunc(),
	},
	"[]time.Duration": {
		"StringToDurationSliceHookFunc",
		StringToDurationSliceHookFunc(),
	},
	"[]bool": {
		"StringToBoolSliceHookFunc",
		StringToBoolSliceHookFunc(),
	},
	"[]uint": {
		"StringToUintSliceHookFunc",
		StringToUintSliceHookFunc(),
	},
	"map[string]string": {
		"StringToStringMapHookFunc",
		StringToStringMapHookFunc(),
	},
	"map[string]int": {
		"StringToIntMapHookFunc",
		StringToIntMapHookFunc(),
	},
	"map[string]int64": {
		"StringToInt64MapHookFunc",
		StringToInt64MapHookFunc(),
	},
	"net.IP": {
		"StringToIPHookFunc",
		mapstructure.StringToIPHookFunc(),
	},
	"net.IPMask": {
		"StringToIPMaskHookFunc",
		StringToIPMaskHookFunc(),
	},
	"net.IPNet": {
		"StringToIPNetHookFunc",
		mapstructure.StringToIPNetHookFunc(),
	},
	"[]net.IP": {
		"StringToIPSliceHookFunc",
		StringToIPSliceHookFunc(),
	},
	"zapcore.Level": {
		"StringToZapcoreLevelHookFunc",
		StringToZapcoreLevelHookFunc(),
	},
	"slog.Level": {
		"StringToSlogLevelHookFunc",
		StringToSlogLevelHookFunc(),
	},
	"[]string": {
		"StringToCSVStringSliceHookFunc",
		StringToCSVStringSliceHookFunc(),
	},
	"[]int": {
		"StringToIntSliceHookFunc",
		StringToIntSliceHookFunc(","),
	},
	"[]uint8": {
		"StringToRawBytesHookFunc",
		StringToRawBytesHookFunc(),
	},
	"structcli.Hex": {
		"StringToHexHookFunc",
		StringToNamedBytesHookFunc("structcli.Hex", decodeHexBytes),
	},
	"structcli.Base64": {
		"StringToBase64HookFunc",
		StringToNamedBytesHookFunc("structcli.Base64", decodeBase64Bytes),
	},
}

// AnnotationToDecodeHookRegistry maps annotation names to decode hook functions
var AnnotationToDecodeHookRegistry map[string]mapstructure.DecodeHookFunc

func init() {
	// Map annotations to decoding hook
	AnnotationToDecodeHookRegistry = make(map[string]mapstructure.DecodeHookFunc)
	for typename, data := range DecodeHookRegistry {
		if _, exists := AnnotationToDecodeHookRegistry[data.ann]; exists {
			panic(fmt.Sprintf("duplicate annotation name '%s' found in decode hook registry (type: %s)", data.ann, typename))
		}

		AnnotationToDecodeHookRegistry[data.ann] = data.fx
	}
}

func InferDecodeHooks(c *cobra.Command, name, typename string) (bool, error) {
	if data, ok := DecodeHookRegistry[typename]; ok {
		if err := c.Flags().SetAnnotation(name, FlagDecodeHookAnnotation, []string{data.ann}); err != nil {
			return false, fmt.Errorf("set decode hook annotation: %w", err)
		}

		return true, nil
	}

	return false, nil
}

// StringToZapcoreLevelHookFunc creates a decode hook that converts string values
// to zapcore.Level types during configuration unmarshaling.
func StringToZapcoreLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(zapcore.DebugLevel) {
			return data, nil
		}

		level, err := zapcore.ParseLevel(data.(string))
		if err != nil {
			return nil, fmt.Errorf("invalid string for zapcore.Level '%s': %w", data.(string), err)
		}

		return level, nil
	}
}

// StringToSlogLevelHookFunc creates a decode hook that converts string values
// to slog.Level types during configuration unmarshaling.
func StringToSlogLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(slog.LevelInfo) {
			return data, nil
		}

		var level slog.Level
		err := level.UnmarshalText([]byte(data.(string)))
		if err != nil {
			return nil, fmt.Errorf("invalid string for slog.Level '%s': %w", data.(string), err)
		}

		return level, nil
	}
}

// StringToIntSliceHookFunc creates a decode hook that converts comma-separated
// string values to []int slices during configuration unmarshaling.
func StringToIntSliceHookFunc(sep string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.SliceOf(reflect.TypeOf(int(0))) {
			return data, nil
		}

		raw := data.(string)
		if raw == "" {
			return []int{}, nil
		}

		parts := strings.Split(raw, sep)
		result := make([]int, len(parts))

		for i, part := range parts {
			trimmed := strings.TrimSpace(part)
			num, err := strconv.Atoi(trimmed)
			if err != nil {
				return nil, fmt.Errorf("invalid integer '%s' at position %d: %w", trimmed, i, err)
			}
			result[i] = num
		}

		return result, nil
	}
}

func parseWithFlagSet[T any](register func(fs *pflag.FlagSet, target *T), raw string) (T, error) {
	var out T
	fs := pflag.NewFlagSet("structcli-decode", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	register(fs, &out)
	if err := fs.Set("value", raw); err != nil {
		return out, err
	}

	return out, nil
}

func normalizePFlagCollectionString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}

	return trimmed
}

func convertMapInput[T any](data any, convertValue func(any) (T, error)) (map[string]T, error) {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Map {
		return nil, fmt.Errorf("invalid map source type %T", data)
	}

	out := make(map[string]T, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		key, ok := iter.Key().Interface().(string)
		if !ok {
			return nil, fmt.Errorf("invalid map key type %T", iter.Key().Interface())
		}

		value, err := convertValue(iter.Value().Interface())
		if err != nil {
			return nil, fmt.Errorf("invalid map value for key %q: %w", key, err)
		}
		out[key] = value
	}

	return out, nil
}

func convertToString(value any) (string, error) {
	return fmt.Sprint(value), nil
}

func convertToInt(value any) (int, error) {
	switch v := value.(type) {
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 0)
		if err != nil {
			return 0, err
		}
		return int(parsed), nil
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func convertToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("overflow converting %d to int64", v)
		}
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func StringToDurationSliceHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf([]time.Duration(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return []time.Duration{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *[]time.Duration) {
				fs.DurationSliceVar(target, "value", nil, "")
			}, raw)
			if err != nil {
				return nil, fmt.Errorf("invalid string for []time.Duration '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Slice, reflect.Array:
			rv := reflect.ValueOf(data)
			out := make([]time.Duration, rv.Len())
			for i := range rv.Len() {
				item := rv.Index(i).Interface()
				switch v := item.(type) {
				case string:
					d, err := time.ParseDuration(strings.TrimSpace(v))
					if err != nil {
						return nil, fmt.Errorf("invalid duration '%s' at position %d: %w", v, i, err)
					}
					out[i] = d
				case time.Duration:
					out[i] = v
				case int:
					out[i] = time.Duration(v)
				case int8:
					out[i] = time.Duration(v)
				case int16:
					out[i] = time.Duration(v)
				case int32:
					out[i] = time.Duration(v)
				case int64:
					out[i] = time.Duration(v)
				case uint:
					out[i] = time.Duration(v)
				case uint8:
					out[i] = time.Duration(v)
				case uint16:
					out[i] = time.Duration(v)
				case uint32:
					out[i] = time.Duration(v)
				case uint64:
					out[i] = time.Duration(v)
				default:
					return nil, fmt.Errorf("invalid element type %T at position %d for []time.Duration", item, i)
				}
			}

			return out, nil
		default:
			return data, nil
		}
	}
}

func StringToBoolSliceHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf([]bool(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return []bool{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *[]bool) {
				fs.BoolSliceVar(target, "value", nil, "")
			}, raw)
			if err != nil {
				return nil, fmt.Errorf("invalid string for []bool '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Slice, reflect.Array:
			rv := reflect.ValueOf(data)
			out := make([]bool, rv.Len())
			for i := range rv.Len() {
				item := rv.Index(i).Interface()
				switch v := item.(type) {
				case string:
					b, err := strconv.ParseBool(strings.TrimSpace(v))
					if err != nil {
						return nil, fmt.Errorf("invalid bool '%s' at position %d: %w", v, i, err)
					}
					out[i] = b
				case bool:
					out[i] = v
				default:
					return nil, fmt.Errorf("invalid element type %T at position %d for []bool", item, i)
				}
			}

			return out, nil
		default:
			return data, nil
		}
	}
}

func StringToUintSliceHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf([]uint(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return []uint{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *[]uint) {
				fs.UintSliceVar(target, "value", nil, "")
			}, raw)
			if err != nil {
				return nil, fmt.Errorf("invalid string for []uint '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Slice, reflect.Array:
			rv := reflect.ValueOf(data)
			out := make([]uint, rv.Len())
			for i := range rv.Len() {
				item := rv.Index(i).Interface()
				switch v := item.(type) {
				case string:
					u, err := strconv.ParseUint(strings.TrimSpace(v), 10, 0)
					if err != nil {
						return nil, fmt.Errorf("invalid uint '%s' at position %d: %w", v, i, err)
					}
					out[i] = uint(u)
				case uint:
					out[i] = v
				case uint8:
					out[i] = uint(v)
				case uint16:
					out[i] = uint(v)
				case uint32:
					out[i] = uint(v)
				case uint64:
					out[i] = uint(v)
				case int:
					if v < 0 {
						return nil, fmt.Errorf("invalid negative uint %d at position %d", v, i)
					}
					out[i] = uint(v)
				case int8:
					if v < 0 {
						return nil, fmt.Errorf("invalid negative uint %d at position %d", v, i)
					}
					out[i] = uint(v)
				case int16:
					if v < 0 {
						return nil, fmt.Errorf("invalid negative uint %d at position %d", v, i)
					}
					out[i] = uint(v)
				case int32:
					if v < 0 {
						return nil, fmt.Errorf("invalid negative uint %d at position %d", v, i)
					}
					out[i] = uint(v)
				case int64:
					if v < 0 {
						return nil, fmt.Errorf("invalid negative uint %d at position %d", v, i)
					}
					out[i] = uint(v)
				default:
					return nil, fmt.Errorf("invalid element type %T at position %d for []uint", item, i)
				}
			}

			return out, nil
		default:
			return data, nil
		}
	}
}

func StringToStringMapHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf(map[string]string(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}
		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return map[string]string{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *map[string]string) {
				fs.StringToStringVar(target, "value", nil, "")
			}, normalizePFlagCollectionString(raw))
			if err != nil {
				return nil, fmt.Errorf("invalid string for map[string]string '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Map:
			return convertMapInput(data, convertToString)
		default:
			return data, nil
		}
	}
}

func StringToIntMapHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf(map[string]int(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}
		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return map[string]int{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *map[string]int) {
				fs.StringToIntVar(target, "value", nil, "")
			}, normalizePFlagCollectionString(raw))
			if err != nil {
				return nil, fmt.Errorf("invalid string for map[string]int '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Map:
			return convertMapInput(data, convertToInt)
		default:
			return data, nil
		}
	}
}

func StringToInt64MapHookFunc() mapstructure.DecodeHookFunc {
	targetType := reflect.TypeOf(map[string]int64(nil))

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t == nil {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != targetType {
			return data, nil
		}
		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			if raw == "" {
				return map[string]int64{}, nil
			}

			out, err := parseWithFlagSet(func(fs *pflag.FlagSet, target *map[string]int64) {
				fs.StringToInt64Var(target, "value", nil, "")
			}, normalizePFlagCollectionString(raw))
			if err != nil {
				return nil, fmt.Errorf("invalid string for map[string]int64 '%s': %w", raw, err)
			}

			return out, nil
		case reflect.Map:
			return convertMapInput(data, convertToInt64)
		default:
			return data, nil
		}
	}
}

// StringToRawBytesHookFunc converts plain textual input into raw []byte.
func StringToRawBytesHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf([]byte{}) {
			return data, nil
		}

		return []byte(data.(string)), nil
	}
}

// StringToNamedBytesHookFunc converts encoded textual input into a named []byte type.
func StringToNamedBytesHookFunc(typeName string, decode func(string) ([]byte, error)) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t.String() != typeName {
			return data, nil
		}

		raw := data.(string)
		decoded, err := decode(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid string for %s '%s': %w", typeName, raw, err)
		}

		return reflect.ValueOf(decoded).Convert(t).Interface(), nil
	}
}

func decodeHexBytes(raw string) ([]byte, error) {
	return hex.DecodeString(strings.TrimSpace(raw))
}

func decodeBase64Bytes(raw string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
}

// StringToIPMaskHookFunc converts textual input into net.IPMask.
func StringToIPMaskHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(net.IPMask(nil)) {
			return data, nil
		}

		raw := data.(string)
		mask := parseIPv4Mask(strings.TrimSpace(raw))
		if mask == nil {
			return nil, fmt.Errorf("invalid string for net.IPMask '%s'", raw)
		}

		return mask, nil
	}
}

// StringToIPSliceHookFunc converts textual and list input into []net.IP.
func StringToIPSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if t != reflect.TypeOf([]net.IP(nil)) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			raw := data.(string)
			ips, err := parseIPSlice(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid string for []net.IP '%s': %w", raw, err)
			}

			return ips, nil
		case reflect.Slice, reflect.Array:
			rv := reflect.ValueOf(data)
			out := make([]net.IP, rv.Len())
			for i := range rv.Len() {
				item := rv.Index(i).Interface()
				switch v := item.(type) {
				case string:
					ip := net.ParseIP(strings.TrimSpace(v))
					if ip == nil {
						return nil, fmt.Errorf("invalid IP '%s' at position %d", v, i)
					}
					out[i] = ip
				case net.IP:
					out[i] = v
				default:
					return nil, fmt.Errorf("invalid element type %T at position %d for []net.IP", item, i)
				}
			}

			return out, nil
		default:
			return data, nil
		}
	}
}

func readAsCSV(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}

	reader := csv.NewReader(strings.NewReader(raw))

	return reader.Read()
}

// StringToCSVStringSliceHookFunc converts textual input into []string using CSV semantics.
func StringToCSVStringSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != reflect.TypeOf([]string(nil)) {
			return data, nil
		}

		raw := data.(string)
		vals, err := readAsCSV(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid string for []string '%s': %w", raw, err)
		}

		return vals, nil
	}
}

func parseIPSlice(raw string) ([]net.IP, error) {
	// Keep CSV splitting semantics aligned with pflag slice parsing:
	// commas split values, quoted items remain intact.
	trimmed := strings.TrimSpace(raw)
	rmQuote := strings.NewReplacer(`"`, "", `'`, "", "`", "")
	trimmed = rmQuote.Replace(trimmed)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	if trimmed == "" {
		return []net.IP{}, nil
	}

	parts, err := readAsCSV(trimmed)
	if err != nil {
		return nil, err
	}

	out := make([]net.IP, len(parts))
	for i, part := range parts {
		ip := net.ParseIP(strings.TrimSpace(part))
		if ip == nil {
			return nil, fmt.Errorf("invalid IP '%s' at position %d", part, i)
		}

		out[i] = ip
	}

	return out, nil
}

func parseIPv4Mask(s string) net.IPMask {
	mask := net.ParseIP(s)
	if mask == nil {
		if len(s) != 8 {
			return nil
		}

		parts := []int{}
		for i := 0; i < 4; i++ {
			b := "0x" + s[2*i:2*i+2]
			d, err := strconv.ParseInt(b, 0, 0)
			if err != nil {
				return nil
			}
			parts = append(parts, int(d))
		}

		s = fmt.Sprintf("%d.%d.%d.%d", parts[0], parts[1], parts[2], parts[3])
		mask = net.ParseIP(s)
		if mask == nil {
			return nil
		}
	}

	return net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
}

func StoreDecodeHookFunc(c *cobra.Command, flagname string, decodeM reflect.Value, target reflect.Type) error {
	s := internalscope.Get(c)

	// Wrap that adapts user method to mapstructure.DecodeHookFuncType signature
	hookFunc := func(from reflect.Type, to reflect.Type, data any) (any, error) {
		// Only apply this hook to the specific target type
		if to != target {
			return data, nil
		}

		// Only convert from string env var and config file values
		// They always come as strings
		if from.Kind() != reflect.String {
			return data, nil
		}

		// Call user's decode hook: DecodeX(input interface{}) (target, error)
		results := decodeM.Call([]reflect.Value{reflect.ValueOf(data)})

		if len(results) != 2 {
			return nil, fmt.Errorf("user decode method must return (value, error)")
		}

		// Check if error is not nil
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		return results[0].Interface(), nil
	}

	k := fmt.Sprintf("customDecodeHook_%s_%s", c.Name(), flagname)
	s.SetCustomDecodeHook(k, hookFunc)

	return c.Flags().SetAnnotation(flagname, FlagDecodeHookAnnotation, []string{k})
}
