package internaltag

import (
	"fmt"
	"strings"
)

// FlagPreset defines a CLI-only flag alias that sets a fixed value
// on a canonical target flag.
type FlagPreset struct {
	Name  string
	Value string
}

// ParseFlagPresets parses a `flagpreset` tag value.
//
// Supported formats:
// - "alias=value"
// - "alias1=value1;alias2=value2"
// - "alias1=value1,alias2=value2"
//
// If both separators are present, ';' wins to allow commas in values.
func ParseFlagPresets(raw string) ([]FlagPreset, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	items := splitPresetEntries(trimmed)
	presets := make([]FlagPreset, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("empty preset alias entry")
		}

		eqIdx := strings.Index(item, "=")
		if eqIdx <= 0 {
			return nil, fmt.Errorf("invalid preset alias '%s': expected <name>=<value>", item)
		}

		name := strings.TrimSpace(item[:eqIdx])
		value := strings.TrimSpace(item[eqIdx+1:])

		if !IsValidFlagName(name) {
			return nil, fmt.Errorf("invalid preset alias name '%s'", name)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate preset alias '%s'", name)
		}
		seen[name] = struct{}{}

		presets = append(presets, FlagPreset{
			Name:  name,
			Value: value,
		})
	}

	return presets, nil
}

func splitPresetEntries(raw string) []string {
	if strings.Contains(raw, ";") {
		return strings.Split(raw, ";")
	}

	return strings.Split(raw, ",")
}
