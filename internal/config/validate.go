package internalconfig

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

// ValidateKeys decodes command-relevant config values into opts' shape and fails
// when unknown keys are present.
func ValidateKeys(configValues map[string]any, opts any, hooks ...mapstructure.DecodeHookFunc) error {
	if len(configValues) == 0 {
		return nil
	}

	target, err := decodeTarget(opts)
	if err != nil {
		return err
	}

	metadata := &mapstructure.Metadata{}
	cfg := &mapstructure.DecoderConfig{
		Result:           target,
		Metadata:         metadata,
		WeaklyTypedInput: true,
	}
	if len(hooks) > 0 {
		cfg.DecodeHook = mapstructure.ComposeDecodeHookFunc(hooks...)
	}

	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return fmt.Errorf("couldn't create config validator: %w", err)
	}
	if err := decoder.Decode(configValues); err != nil {
		return err
	}

	if len(metadata.Unused) == 0 {
		return nil
	}

	knownKeys := knownConfigKeys(reflect.TypeOf(opts))
	unknown := make([]string, 0, len(metadata.Unused))
	for _, key := range metadata.Unused {
		norm := strings.ToLower(key)
		if _, ok := knownKeys[norm]; ok {
			continue
		}
		unknown = append(unknown, norm)
	}
	if len(unknown) == 0 {
		return nil
	}

	unique := dedupe(unknown)
	sort.Strings(unique)

	return fmt.Errorf("unknown config keys: %s", strings.Join(unique, ", "))
}

func decodeTarget(opts any) (any, error) {
	T := reflect.TypeOf(opts)
	if T == nil {
		return nil, fmt.Errorf("couldn't validate config: nil options target")
	}
	if T.Kind() == reflect.Ptr {
		T = T.Elem()
	}
	if T.Kind() != reflect.Struct {
		return nil, fmt.Errorf("couldn't validate config: options target must be a struct")
	}

	return reflect.New(T).Interface(), nil
}

func dedupe(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}

	return out
}

func knownConfigKeys(T reflect.Type) map[string]struct{} {
	out := make(map[string]struct{})
	collectKnownConfigKeys(T, "", out)

	return out
}

func collectKnownConfigKeys(T reflect.Type, prefix string, out map[string]struct{}) {
	if T == nil {
		return
	}
	if T.Kind() == reflect.Ptr {
		T = T.Elem()
	}
	if T.Kind() != reflect.Struct {
		return
	}

	for i := range T.NumField() {
		f := T.Field(i)
		if !f.IsExported() {
			continue
		}

		fieldName := strings.ToLower(f.Name)
		out[prefix+fieldName] = struct{}{}

		alias := strings.ToLower(f.Tag.Get("flag"))
		if alias != "" {
			out[prefix+alias] = struct{}{}
		}

		nestedType := f.Type
		if nestedType.Kind() == reflect.Ptr {
			nestedType = nestedType.Elem()
		}
		if nestedType.Kind() == reflect.Struct {
			collectKnownConfigKeys(nestedType, prefix+fieldName+".", out)
			if alias != "" && alias != fieldName {
				collectKnownConfigKeys(nestedType, prefix+alias+".", out)
			}
		}
	}
}
