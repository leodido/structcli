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
)

const (
	FlagDecodeHookAnnotation = "leodido/structcli/flag-decode-hooks"
)

type DecodeHookFunc func(input any) (any, error)

type decodingAnnotation struct {
	ann string
	fx  mapstructure.DecodeHookFunc
}

// DecodeHookRegistry maps reflect.Type to decode hook metadata.
// Keyed by reflect.Type for collision-safe lookups.
var DecodeHookRegistry = map[reflect.Type]decodingAnnotation{
	reflect.TypeFor[time.Duration](): {
		"StringToTimeDurationHookFunc",
		mapstructure.StringToTimeDurationHookFunc(),
	},
	reflect.TypeFor[[]time.Duration](): {
		"StringToDurationSliceHookFunc",
		StringToDurationSliceHookFunc(),
	},
	reflect.TypeFor[[]bool](): {
		"StringToBoolSliceHookFunc",
		StringToBoolSliceHookFunc(),
	},
	reflect.TypeFor[[]uint](): {
		"StringToUintSliceHookFunc",
		StringToUintSliceHookFunc(),
	},
	reflect.TypeFor[map[string]string](): {
		"StringToStringMapHookFunc",
		StringToStringMapHookFunc(),
	},
	reflect.TypeFor[map[string]int](): {
		"StringToIntMapHookFunc",
		StringToIntMapHookFunc(),
	},
	reflect.TypeFor[map[string]int64](): {
		"StringToInt64MapHookFunc",
		StringToInt64MapHookFunc(),
	},
	reflect.TypeFor[net.IP](): {
		"StringToIPHookFunc",
		mapstructure.StringToIPHookFunc(),
	},
	reflect.TypeFor[net.IPMask](): {
		"StringToIPMaskHookFunc",
		StringToIPMaskHookFunc(),
	},
	reflect.TypeFor[net.IPNet](): {
		"StringToIPNetHookFunc",
		mapstructure.StringToIPNetHookFunc(),
	},
	reflect.TypeFor[[]net.IP](): {
		"StringToIPSliceHookFunc",
		StringToIPSliceHookFunc(),
	},
	reflect.TypeFor[slog.Level](): {
		"StringToSlogLevelHookFunc",
		StringToSlogLevelHookFunc(),
	},
	reflect.TypeFor[[]string](): {
		"StringToCSVStringSliceHookFunc",
		StringToCSVStringSliceHookFunc(),
	},
	reflect.TypeFor[[]int](): {
		"StringToIntSliceHookFunc",
		StringToIntSliceHookFunc(","),
	},
	reflect.TypeFor[[]uint8](): {
		"StringToRawBytesHookFunc",
		StringToRawBytesHookFunc(),
	},
}

// AnnotationToDecodeHookRegistry maps annotation names to decode hook functions
var AnnotationToDecodeHookRegistry map[string]mapstructure.DecodeHookFunc

func init() {
	// Map annotations to decoding hook
	AnnotationToDecodeHookRegistry = make(map[string]mapstructure.DecodeHookFunc)
	for typ, data := range DecodeHookRegistry {
		if _, exists := AnnotationToDecodeHookRegistry[data.ann]; exists {
			panic(fmt.Sprintf("duplicate annotation name '%s' found in decode hook registry (type: %s)", data.ann, typ))
		}

		AnnotationToDecodeHookRegistry[data.ann] = data.fx
	}
}

func InferDecodeHooks(c *cobra.Command, name string, typ reflect.Type) bool {
	if data, ok := DecodeHookRegistry[typ]; ok {
		if err := c.Flags().SetAnnotation(name, FlagDecodeHookAnnotation, []string{data.ann}); err != nil {
			panic(fmt.Sprintf("structcli: SetAnnotation on just-registered flag %q: %v", name, err))
		}

		return true
	}

	return false
}

// DecodeRegistrySnapshot holds opaque copies of both decode registries for
// test isolation.
type DecodeRegistrySnapshot struct {
	registry    map[reflect.Type]decodingAnnotation
	annotations map[string]mapstructure.DecodeHookFunc
}

// SnapshotDecodeRegistries returns a deep copy of both decode registries.
func SnapshotDecodeRegistries() DecodeRegistrySnapshot {
	dr := make(map[reflect.Type]decodingAnnotation, len(DecodeHookRegistry))
	for k, v := range DecodeHookRegistry {
		dr[k] = v
	}
	ar := make(map[string]mapstructure.DecodeHookFunc, len(AnnotationToDecodeHookRegistry))
	for k, v := range AnnotationToDecodeHookRegistry {
		ar[k] = v
	}

	return DecodeRegistrySnapshot{registry: dr, annotations: ar}
}

// RestoreDecodeRegistries replaces both decode registries from a snapshot.
func RestoreDecodeRegistries(snap DecodeRegistrySnapshot) {
	DecodeHookRegistry = snap.registry
	AnnotationToDecodeHookRegistry = snap.annotations
}

// RegisterDecodeHook registers a decode hook for a custom type. It updates both
// DecodeHookRegistry and AnnotationToDecodeHookRegistry. Panics on duplicate
// annotation name (consistent with init() behavior).
func RegisterDecodeHook(typ reflect.Type, annotationName string, hook mapstructure.DecodeHookFunc) {
	if _, exists := AnnotationToDecodeHookRegistry[annotationName]; exists {
		panic(fmt.Sprintf("duplicate annotation name '%s' in decode hook registry (type: %s)", annotationName, typ))
	}

	DecodeHookRegistry[typ] = decodingAnnotation{ann: annotationName, fx: hook}
	AnnotationToDecodeHookRegistry[annotationName] = hook
}

// RegisterUserDecodeHook wraps a user-provided DecodeHookFunc into a
// mapstructure.DecodeHookFunc and registers it for the given type.
// The wrapper filters by target type (reflect.Type equality) and source kind (string).
func RegisterUserDecodeHook(typ reflect.Type, decode DecodeHookFunc) {
	annName := fmt.Sprintf("RegisterTypeTo%sHookFunc", sanitizeTypeName(typ.String()))

	hook := func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to != typ {
			return data, nil
		}
		if from.Kind() != reflect.String {
			return data, nil
		}

		return decode(data)
	}

	RegisterDecodeHook(typ, annName, mapstructure.DecodeHookFunc(hook))
}

func sanitizeTypeName(typeName string) string {
	r := strings.NewReplacer(
		".", "_",
		"[", "_",
		"]", "",
		"*", "Ptr",
		"/", "_",
	)

	return r.Replace(typeName)
}

// StringToEnumHookFunc creates a decode hook that converts string values to a
// ~string enum type during configuration unmarshaling. It supports
// case-insensitive matching and aliases.
func StringToEnumHookFunc[E ~string](values map[E][]string) mapstructure.DecodeHookFunc {
	allowed := make(map[string]E)
	for enumVal, names := range values {
		for _, name := range names {
			key := strings.ToLower(name)
			if existing, ok := allowed[key]; ok {
				panic(fmt.Sprintf("structcli: alias %q (lowercased) maps to both %v and %v", key, existing, enumVal))
			}
			allowed[key] = enumVal
		}
	}
	targetType := reflect.TypeFor[E]()

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t != targetType {
			return data, nil
		}
		s, ok := data.(string)
		if !ok {
			return data, nil
		}
		if val, found := allowed[strings.ToLower(s)]; found {
			return val, nil
		}

		return nil, fmt.Errorf("invalid value %q for %s", s, targetType.Name())
	}
}

// StringToIntEnumHookFunc creates a decode hook that converts string values to
// an integer-based enum type during configuration unmarshaling. It supports
// case-insensitive matching and aliases.
func StringToIntEnumHookFunc[E ~int | ~int8 | ~int16 | ~int32 | ~int64](values map[E][]string) mapstructure.DecodeHookFunc {
	allowed := make(map[string]E)
	for enumVal, names := range values {
		for _, name := range names {
			key := strings.ToLower(name)
			if existing, ok := allowed[key]; ok {
				panic(fmt.Sprintf("structcli: alias %q (lowercased) maps to both %v and %v", key, existing, enumVal))
			}
			allowed[key] = enumVal
		}
	}
	targetType := reflect.TypeFor[E]()

	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != targetType {
			return data, nil
		}
		if f.Kind() != reflect.String {
			return data, nil
		}
		s, ok := data.(string)
		if !ok {
			return data, nil
		}
		if val, found := allowed[strings.ToLower(s)]; found {
			return val, nil
		}

		return nil, fmt.Errorf("invalid value %q for %s", s, targetType.Name())
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
// Matches by reflect.Type equality for collision safety.
func StringToNamedBytesHookFunc(targetType reflect.Type, decode func(string) ([]byte, error)) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != targetType {
			return data, nil
		}

		raw := data.(string)
		decoded, err := decode(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid string for %s '%s': %w", targetType, raw, err)
		}

		return reflect.ValueOf(decoded).Convert(t).Interface(), nil
	}
}

// DecodeHexBytes decodes a hex-encoded string into bytes.
func DecodeHexBytes(raw string) ([]byte, error) {
	return hex.DecodeString(strings.TrimSpace(raw))
}

// DecodeBase64Bytes decodes a base64-encoded string into bytes.
func DecodeBase64Bytes(raw string) ([]byte, error) {
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

// StoreDecodeHookFuncDirect registers a typed decode hook for a flag.
func StoreDecodeHookFuncDirect(c *cobra.Command, flagname string, decode DecodeHookFunc, target reflect.Type) {
	s := internalscope.Get(c)

	hookFunc := func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to != target {
			return data, nil
		}
		if from.Kind() != reflect.String {
			return data, nil
		}

		return decode(data)
	}

	k := fmt.Sprintf("customDecodeHook_%s_%s", c.Name(), flagname)
	s.SetCustomDecodeHook(k, hookFunc)

	if err := c.Flags().SetAnnotation(flagname, FlagDecodeHookAnnotation, []string{k}); err != nil {
		panic(fmt.Sprintf("structcli: SetAnnotation on just-registered flag %q: %v", flagname, err))
	}
}
