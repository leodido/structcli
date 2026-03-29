package structcli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	internaldebug "github.com/leodido/structcli/internal/debug"
	internalenv "github.com/leodido/structcli/internal/env"
	internalusage "github.com/leodido/structcli/internal/usage"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// enumPattern matches the {val1,val2,...} pattern in flag usage strings.
// Values must be simple identifiers (alphanumeric, hyphens, underscores) — this avoids
// matching config search path patterns like {/etc/app,...} or similar non-enum braces.
var enumPattern = regexp.MustCompile(`\{([\w-]+(?:,[\w-]+)+)\}`)

// PresetInfo describes a preset alias for a flag.
type PresetInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FlagSchema describes a single flag in machine-readable form.
type FlagSchema struct {
	Name        string       `json:"name"`
	Shorthand   string       `json:"shorthand,omitempty"`
	Type        string       `json:"type"`
	Default     string       `json:"default,omitempty"`
	Description string       `json:"description,omitempty"`
	Required    bool         `json:"required,omitempty"`
	EnvVars     []string     `json:"env_vars,omitempty"`
	Group       string       `json:"group,omitempty"`
	FieldPath   string       `json:"field_path,omitempty"`
	Enum        []string     `json:"enum,omitempty"`
	Presets     []PresetInfo `json:"presets,omitempty"`
}

// CommandSchema describes a command's inputs in machine-readable form.
type CommandSchema struct {
	Name        string                 `json:"name"`
	CommandPath string                 `json:"command_path"`
	Description string                 `json:"description,omitempty"`
	Flags       map[string]*FlagSchema `json:"flags"`
	Groups      map[string][]string    `json:"groups,omitempty"`
	Subcommands []string               `json:"subcommands,omitempty"`
	EnvPrefix   string                 `json:"env_prefix,omitempty"`
}

// JSONSchema returns machine-readable schemas for a command's inputs.
//
// By default it returns a single-element slice with the schema for the given command.
// Pass jsonschema.WithFullTree() to walk the entire command tree and return schemas for all subcommands.
//
// It extracts all flag metadata from cobra annotations set during Define(),
// including types, defaults, descriptions, environment variables, groups, presets, and enum values.
func JSONSchema(c *cobra.Command, opts ...jsonschema.Opt) ([]*CommandSchema, error) {
	cfg := jsonschema.Apply(opts...)

	if cfg.FullTree {
		return jsonSchemaTree(c, cfg)
	}

	schema, err := jsonSchemaOne(c, cfg)
	if err != nil {
		return nil, err
	}

	return []*CommandSchema{schema}, nil
}

// jsonSchemaOne returns the schema for a single command.
func jsonSchemaOne(c *cobra.Command, cfg *jsonschema.Config) (*CommandSchema, error) {
	schema := &CommandSchema{
		Name:        c.Name(),
		CommandPath: c.CommandPath(),
		Description: c.Short,
		Flags:       make(map[string]*FlagSchema),
		EnvPrefix:   EnvPrefix(),
	}

	if c.Long != "" {
		schema.Description = c.Long
	}

	// Collect groups
	groups := make(map[string][]string)

	// Resolve the jsonschema flag name so we can skip it in output
	jsonSchemaFlagName := ""
	if rootAnnotations := c.Root().Annotations; rootAnnotations != nil {
		jsonSchemaFlagName = rootAnnotations[jsonSchemaFlagAnnotation]
	}

	// Walk all flags (local + inherited)
	c.Flags().VisitAll(func(f *pflag.Flag) {
		// Skip hidden and deprecated flags
		if f.Hidden || f.Deprecated != "" {
			return
		}

		// Skip cobra built-in help flag and structcli meta-flags
		if f.Name == "help" || f.Name == jsonSchemaFlagName {
			return
		}

		// Skip structcli infrastructure flags (debug, config)
		if rootAnnotations := c.Root().Annotations; rootAnnotations != nil {
			if f.Name == rootAnnotations[internaldebug.FlagAnnotation] ||
				f.Name == rootAnnotations[configFlagAnnotation] {
				return
			}
		}

		// Skip preset alias flags (they are represented via x-structcli-presets on the target flag)
		if strings.HasPrefix(f.Usage, "alias for --") {
			return
		}

		fs := &FlagSchema{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
		}

		// Extract enum values: prefer the machine-readable annotation set during
		// Define(), falling back to regex extraction from the usage string for
		// flags that were not created by structcli (e.g. manually added flags).
		descr := f.Usage
		if enumMetadata, ok := f.Annotations[flagEnumAnnotation]; ok && len(enumMetadata) > 0 {
			fs.Enum = enumMetadata
			if !cfg.EnumInDescription {
				descr = strings.TrimSpace(enumPattern.ReplaceAllString(descr, ""))
			}
		} else if matches := enumPattern.FindStringSubmatch(descr); len(matches) > 1 {
			vals := strings.Split(matches[1], ",")
			for i := range vals {
				vals[i] = strings.TrimSpace(vals[i])
			}
			fs.Enum = vals
			if !cfg.EnumInDescription {
				descr = strings.TrimSpace(enumPattern.ReplaceAllString(descr, ""))
			}
		}
		fs.Description = descr

		// Check required via cobra's annotation
		if requiredAnnotation, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok && len(requiredAnnotation) > 0 {
			if requiredAnnotation[0] == "true" {
				fs.Required = true
			}
		}

		// Read default from structcli annotation (more reliable than pflag DefValue for custom types)
		if defaultMetadata, ok := f.Annotations[flagDefaultAnnotation]; ok && len(defaultMetadata) > 0 {
			fs.Default = defaultMetadata[0]
		}

		// Read field path
		if pathMetadata, ok := f.Annotations[flagPathAnnotation]; ok && len(pathMetadata) > 0 {
			fs.FieldPath = pathMetadata[0]
		}

		// Read environment variables
		if envMetadata, ok := f.Annotations[internalenv.FlagAnnotation]; ok && len(envMetadata) > 0 {
			fs.EnvVars = envMetadata
		}

		// Read group
		if groupMetadata, ok := f.Annotations[internalusage.FlagGroupAnnotation]; ok && len(groupMetadata) > 0 {
			fs.Group = groupMetadata[0]
			groups[groupMetadata[0]] = append(groups[groupMetadata[0]], f.Name)
		}

		// Read presets
		if presetMetadata, ok := f.Annotations[flagPresetsAnnotation]; ok && len(presetMetadata) > 0 {
			fs.Presets = make([]PresetInfo, 0, len(presetMetadata))
			for _, entry := range presetMetadata {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) == 2 {
					fs.Presets = append(fs.Presets, PresetInfo{Name: parts[0], Value: parts[1]})
				}
			}
		}

		schema.Flags[f.Name] = fs
	})

	if len(groups) > 0 {
		schema.Groups = groups
	}

	// Collect subcommand names
	for _, sub := range c.Commands() {
		if !sub.IsAvailableCommand() && sub.Name() != "help" {
			continue
		}
		schema.Subcommands = append(schema.Subcommands, sub.Name())
	}

	return schema, nil
}

// jsonSchemaTree walks the command tree depth-first and returns schemas for all available commands.
func jsonSchemaTree(rootC *cobra.Command, cfg *jsonschema.Config) ([]*CommandSchema, error) {
	var schemas []*CommandSchema

	var walk func(c *cobra.Command) error
	walk = func(c *cobra.Command) error {
		schema, err := jsonSchemaOne(c, cfg)
		if err != nil {
			return err
		}
		schemas = append(schemas, schema)

		for _, sub := range c.Commands() {
			if !sub.IsAvailableCommand() && sub.Name() != "help" {
				continue
			}
			if err := walk(sub); err != nil {
				return err
			}
		}

		return nil
	}

	if err := walk(rootC); err != nil {
		return nil, err
	}

	return schemas, nil
}

// jsonSchemaProperty represents a property in JSON Schema.
type jsonSchemaProperty struct {
	Type        string      `json:"type,omitempty"`
	Default     any         `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Items       *jsonSchema `json:"items,omitempty"`

	// x-structcli extensions
	EnvVars   []string     `json:"x-structcli-env-vars,omitempty"`
	Shorthand string       `json:"x-structcli-shorthand,omitempty"`
	Group     string       `json:"x-structcli-group,omitempty"`
	FieldPath string       `json:"x-structcli-field-path,omitempty"`
	Presets   []PresetInfo `json:"x-structcli-presets,omitempty"`
}

// jsonSchema is a JSON Schema document.
type jsonSchema struct {
	Schema      string                         `json:"$schema,omitempty"`
	Title       string                         `json:"title,omitempty"`
	Description string                         `json:"description,omitempty"`
	Type        string                         `json:"type,omitempty"`
	Properties  map[string]*jsonSchemaProperty `json:"properties,omitempty"`
	Required    []string                       `json:"required,omitempty"`

	// x-structcli extensions at root level
	Subcommands []string            `json:"x-structcli-subcommands,omitempty"`
	EnvPrefix   string              `json:"x-structcli-env-prefix,omitempty"`
	Groups      map[string][]string `json:"x-structcli-groups,omitempty"`
}

// pflagTypeToJSONSchemaType maps pflag type names to JSON Schema types.
func pflagTypeToJSONSchemaType(pflagType string) (string, *jsonSchema) {
	switch pflagType {
	case "bool":
		return "boolean", nil
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "count":
		return "integer", nil
	case "float32", "float64":
		return "number", nil
	case "string", "duration", "zapcore.Level", "slog.Level",
		"ip", "ipMask", "ipNet":
		return "string", nil
	case "stringSlice", "intSlice", "uintSlice", "durationSlice", "boolSlice",
		"ipSlice":
		itemType := "string"
		switch pflagType {
		case "intSlice", "uintSlice":
			itemType = "integer"
		case "boolSlice":
			itemType = "boolean"
		}
		return "array", &jsonSchema{Type: itemType}
	case "stringToString", "stringToInt", "stringToInt64":
		return "object", nil
	case "hexBytes", "base64Bytes", "bytesBase64", "bytesHex":
		return "string", nil
	default:
		return "string", nil
	}
}

// typedDefault converts a string default value to a typed value for JSON Schema.
func typedDefault(defval string, jsonType string, items *jsonSchema) any {
	if defval == "" {
		return nil
	}
	switch jsonType {
	case "boolean":
		return defval == "true"
	case "integer":
		return json.Number(defval)
	case "number":
		// Use json.Number to preserve integer/float precision in the output
		return json.Number(defval)
	case "array":
		// Split comma-separated defaults into a JSON array with typed items.
		if items == nil || items.Type == "" {
			if defval == "[]" {
				return []string{}
			}
			parts := strings.Split(defval, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return parts
		}
		if defval == "[]" {
			switch items.Type {
			case "boolean":
				return []bool{}
			case "integer", "number":
				return []json.Number{}
			default:
				return []string{}
			}
		}
		parts := strings.Split(defval, ",")
		switch items.Type {
		case "boolean":
			result := make([]bool, 0, len(parts))
			for _, part := range parts {
				result = append(result, strings.EqualFold(strings.TrimSpace(part), "true"))
			}
			return result
		case "integer", "number":
			result := make([]json.Number, 0, len(parts))
			for _, part := range parts {
				result = append(result, json.Number(strings.TrimSpace(part)))
			}
			return result
		default:
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return parts
		}
	default:
		return defval
	}
}

// ToJSONSchema converts a CommandSchema to a JSON Schema draft 2020-12 document.
//
// Standard JSON Schema fields (type, properties, required, enum, default, description)
// are used for core flag metadata. structcli-specific metadata is preserved in
// x-structcli-* extension fields.
func (cs *CommandSchema) ToJSONSchema() ([]byte, error) {
	schema := &jsonSchema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		Title:       cs.CommandPath,
		Description: cs.Description,
		Type:        "object",
		Properties:  make(map[string]*jsonSchemaProperty),
	}

	if cs.EnvPrefix != "" {
		schema.EnvPrefix = cs.EnvPrefix
	}
	if len(cs.Subcommands) > 0 {
		schema.Subcommands = cs.Subcommands
	}
	if len(cs.Groups) > 0 {
		schema.Groups = cs.Groups
	}

	var required []string
	for flagName, fs := range cs.Flags {
		jsonType, items := pflagTypeToJSONSchemaType(fs.Type)

		prop := &jsonSchemaProperty{
			Type:        jsonType,
			Description: fs.Description,
			Items:       items,
		}

		if def := typedDefault(fs.Default, jsonType, items); def != nil {
			prop.Default = def
		}
		if len(fs.Enum) > 0 {
			prop.Enum = fs.Enum
		}
		if fs.Shorthand != "" {
			prop.Shorthand = fs.Shorthand
		}
		if len(fs.EnvVars) > 0 {
			prop.EnvVars = fs.EnvVars
		}
		if fs.Group != "" {
			prop.Group = fs.Group
		}
		if fs.FieldPath != "" {
			prop.FieldPath = fs.FieldPath
		}
		if len(fs.Presets) > 0 {
			prop.Presets = fs.Presets
		}

		schema.Properties[flagName] = prop

		if fs.Required {
			required = append(required, flagName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return json.MarshalIndent(schema, "", "  ")
}

// SetupJSONSchema adds a --jsonschema persistent flag to the root command.
//
// When the flag is set, the command prints its JSON Schema to stdout and exits.
// Works only for the root command.
func SetupJSONSchema(rootC *cobra.Command, opts jsonschema.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupJSONSchema must be called on the root command")
	}

	flagName := opts.FlagName
	if flagName == "" {
		flagName = "jsonschema"
	}
	schemaOpts := opts.SchemaOpts
	cfg := jsonschema.Apply(schemaOpts...)

	rootC.PersistentFlags().Bool(flagName, false, "output JSON Schema for this command and exit")

	// Store the flag name in root annotations for lookup
	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	rootC.Annotations[jsonSchemaFlagAnnotation] = flagName

	// Wrap the root persistent pre-run so it can render schema output before the
	// command's normal execution path. This preserves any pre-existing handler.
	wrapForJSONSchema(rootC, flagName, cfg)

	// Regenerate usage templates
	SetupUsage(rootC)

	return nil
}

const jsonSchemaFlagAnnotation = "___leodido_structcli_jsonschemaflagname"

// renderJSONSchemaIfRequested renders schema output when the setup flag is set.
// It returns handled=true when the schema was rendered and the caller should stop.
func renderJSONSchemaIfRequested(c *cobra.Command, flagName string, cfg *jsonschema.Config) (bool, []byte, error) {
	flagSets := []*pflag.FlagSet{
		c.Flags(),
		c.InheritedFlags(),
		c.Root().PersistentFlags(),
	}

	flagChanged := false
	for _, fs := range flagSets {
		if fs == nil {
			continue
		}
		if flag := fs.Lookup(flagName); flag != nil && flag.Changed {
			flagChanged = true
			break
		}
	}

	if !flagChanged {
		return false, nil, nil
	}

	schemas, err := JSONSchema(c, schemaOptsFromConfig(cfg)...)
	if err != nil {
		return true, nil, fmt.Errorf("couldn't generate JSON Schema: %w", err)
	}

	if len(schemas) == 0 {
		return true, nil, fmt.Errorf("couldn't generate JSON Schema: no schemas produced")
	}

	if len(schemas) == 1 {
		output, err := schemas[0].ToJSONSchema()
		if err != nil {
			return true, nil, fmt.Errorf("couldn't generate JSON Schema: %w", err)
		}

		return true, output, nil
	}

	outputs := make([]json.RawMessage, 0, len(schemas))
	for _, schema := range schemas {
		output, err := schema.ToJSONSchema()
		if err != nil {
			return true, nil, fmt.Errorf("couldn't generate JSON Schema: %w", err)
		}
		outputs = append(outputs, json.RawMessage(output))
	}

	output, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return true, nil, fmt.Errorf("couldn't generate JSON Schema: %w", err)
	}

	return true, output, nil
}

func schemaOptsFromConfig(cfg *jsonschema.Config) []jsonschema.Opt {
	if cfg == nil {
		return nil
	}

	var opts []jsonschema.Opt
	if cfg.FullTree {
		opts = append(opts, jsonschema.WithFullTree())
	}
	if cfg.EnumInDescription {
		opts = append(opts, jsonschema.WithEnumInDescription())
	}

	return opts
}

// wrapForJSONSchema recursively wraps commands to intercept --jsonschema.
func wrapForJSONSchema(c *cobra.Command, flagName string, cfg *jsonschema.Config) {
	originalPreRun := c.PersistentPreRunE
	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		handled, output, err := renderJSONSchemaIfRequested(cmd, flagName, cfg)
		if err != nil {
			return err
		}
		if handled {
			fmt.Fprintln(os.Stdout, string(output))
			os.Exit(0)
		}

		if originalPreRun != nil {
			return originalPreRun(cmd, args)
		}

		return nil
	}
}
