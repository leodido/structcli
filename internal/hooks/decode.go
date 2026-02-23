package internalhooks

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
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
		"StringToSliceHookFunc",
		mapstructure.StringToSliceHookFunc(","),
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

func InferDecodeHooks(c *cobra.Command, name, typename string) bool {
	if data, ok := DecodeHookRegistry[typename]; ok {
		_ = c.Flags().SetAnnotation(name, FlagDecodeHookAnnotation, []string{data.ann})

		return true
	}

	return false
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
