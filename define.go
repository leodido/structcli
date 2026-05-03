package structcli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	internalenv "github.com/leodido/structcli/internal/env"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalpath "github.com/leodido/structcli/internal/path"
	internalreflect "github.com/leodido/structcli/internal/reflect"
	internaltag "github.com/leodido/structcli/internal/tag"
	internalusage "github.com/leodido/structcli/internal/usage"
	internalvalidation "github.com/leodido/structcli/internal/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// mustSetAnnotation panics if SetAnnotation fails. The only failure mode is
// a missing flag, which is structurally impossible when called immediately
// after flag registration.
func mustSetAnnotation(fs *pflag.FlagSet, name, key string, values []string) {
	if err := fs.SetAnnotation(name, key, values); err != nil {
		panic(fmt.Sprintf("structcli: SetAnnotation(%q, %q) on just-registered flag: %v", name, key, err))
	}
}

// mustMarkHidden panics if MarkHidden fails (same invariant as mustSetAnnotation).
func mustMarkHidden(fs *pflag.FlagSet, name string) {
	if err := fs.MarkHidden(name); err != nil {
		panic(fmt.Sprintf("structcli: MarkHidden(%q) on just-registered flag: %v", name, err))
	}
}

// mustMarkRequired panics if MarkFlagRequired fails (same invariant).
// Note: MarkFlagRequired is on *cobra.Command, not *pflag.FlagSet.
func mustMarkRequired(c *cobra.Command, name string) {
	if err := c.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("structcli: MarkFlagRequired(%q) on just-registered flag: %v", name, err))
	}
}

// DefineOption configures the behavior of the Define function.
type DefineOption func(*defineContext)

const (
	// DefaultValidateTagName is the struct tag name for validation rules.
	// Matches the go-playground/validator default. Exported so callers
	// configuring their own validator can reference the structcli default
	// (e.g. validator.New().SetTagName(structcli.DefaultValidateTagName)).
	DefaultValidateTagName = "validate"

	// DefaultModTagName is the struct tag name for transformation rules.
	// Matches the go-playground/mold default. Exported so callers
	// configuring their own mold can reference the structcli default.
	DefaultModTagName = "mod"
)

// defineContext holds context for the definition of the options
type defineContext struct {
	exclusions      map[string]string
	comm            *cobra.Command
	validateTagName string // defaults to DefaultValidateTagName
	modTagName      string // defaults to DefaultModTagName
}

// WithValidateTagName sets the struct tag name used to read validation rules.
//
// Defaults to DefaultValidateTagName ("validate", the go-playground/validator default).
// Use this when your validator is configured with a custom tag name
// (eg. validator.New().SetTagName("binding")).
func WithValidateTagName(name string) DefineOption {
	return func(cfg *defineContext) {
		cfg.validateTagName = name
	}
}

// WithModTagName sets the struct tag name used to read transformation rules.
//
// Defaults to DefaultModTagName ("mod", the go-playground/mold default).
// Use this when your mold instance is configured with a custom tag name.
func WithModTagName(name string) DefineOption {
	return func(cfg *defineContext) {
		cfg.modTagName = name
	}
}

// WithExclusions sets flags to exclude from definition based on flag names or paths.
//
// Exclusions are case-insensitive and apply only to the specific command.
func WithExclusions(exclusions ...string) DefineOption {
	return func(cfg *defineContext) {
		if cfg.exclusions == nil {
			cfg.exclusions = make(map[string]string)
		}
		// Map exclusions to the command name
		for _, flag := range exclusions {
			cfg.exclusions[strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(flag), "-"), "-")] = cfg.comm.Name()
		}
	}
}

// Define creates flags from struct field tags and binds them to the command.
//
// It processes struct tags to generate appropriate cobra flags, handles environment
// variable binding, sets up flag groups, and configures the usage template.
func Define(c *cobra.Command, o Options, defineOpts ...DefineOption) error {
	ctx := &defineContext{
		comm: c,
	}

	// Apply configuration options
	for _, opt := range defineOpts {
		opt(ctx)
	}

	// Apply defaults for tag names
	if ctx.validateTagName == "" {
		ctx.validateTagName = DefaultValidateTagName
	}
	if ctx.modTagName == "" {
		ctx.modTagName = DefaultModTagName
	}

	// Run input validation (on by default)
	if err := internalvalidation.Struct(c, o); err != nil {
		return err
	}

	v := GetViper(c)

	// Define the flags from struct
	if err := define(c, o, "", "", ctx.exclusions, false, false, ctx.validateTagName, ctx.modTagName); err != nil {
		return err
	}
	// Bind flag values to struct field values
	v.BindPFlags(c.Flags())
	// Bind environment
	if err := internalenv.BindEnv(c); err != nil {
		return fmt.Errorf("couldn't bind environment variables: %w", err)
	}
	// Generate the usage message
	SetupUsage(c)

	return nil
}

func define(c *cobra.Command, o any, startingGroup string, structPath string, exclusions map[string]string, defineEnv bool, mandatory bool, validateTagName string, modTagName string) error {
	// Assuming validation already caught untyped nils...
	val := internalreflect.GetValue(o)
	if !val.IsValid() {
		val = internalreflect.GetValue(internalreflect.GetValuePtr(o).Interface())
	}
	// Resolve per-field hooks from interfaces.
	var fieldHooks map[string]FieldHook
	if fhp, ok := o.(FieldHookProvider); ok {
		fieldHooks = fhp.FieldHooks()
	}
	var completionHooks map[string]CompleteHookFunc
	if fc, ok := o.(FieldCompleter); ok {
		completionHooks = fc.CompletionHooks()
	}

	// Validate hook map keys against actual struct fields.
	if len(fieldHooks) > 0 || len(completionHooks) > 0 {
		fieldNames := make(map[string]bool, val.NumField())
		for i := range val.NumField() {
			fieldNames[val.Type().Field(i).Name] = true
		}
		for key, fh := range fieldHooks {
			if !fieldNames[key] {
				return fmt.Errorf("FieldHookProvider: key %q does not match any struct field in %s", key, val.Type().Name())
			}
			if fh.Define == nil && fh.Decode != nil {
				return fmt.Errorf("FieldHookProvider: key %q has Decode without Define (Define is required when Decode is set)", key)
			}
		}
		for key := range completionHooks {
			if !fieldNames[key] {
				return fmt.Errorf("FieldCompleter: key %q does not match any struct field in %s", key, val.Type().Name())
			}
		}
	}

	for i := range val.NumField() {
		field := val.Field(i)
		f := val.Type().Field(i)

		// Ignore unexported fields, but recurse into unexported embedded structs
		// because their exported fields are promoted and accessible.
		if !field.CanInterface() {
			if f.Anonymous && f.Type.Kind() == reflect.Struct && field.CanAddr() {
				// Use reflect.NewAt to obtain an interfaceable pointer to the
				// unexported embedded struct so we can pass it to the recursive call.
				ptr := reflect.NewAt(f.Type, field.Addr().UnsafePointer())
				if err := define(c, ptr.Interface(), startingGroup, structPath, exclusions, defineEnv, mandatory, validateTagName, modTagName); err != nil {
					return err
				}
			}
			continue
		}

		// Only if the field is addressable
		if !field.CanAddr() {
			continue
		}
		path := internalpath.GetFieldPath(structPath, f)

		// Check exclusions for struct path with command name validation (case-insensitive)
		if cname, ok := exclusions[path]; ok && c.Name() == cname {
			continue
		}

		// Check exclusions for alias with command name validation (case-insensitive)
		alias := f.Tag.Get("flag")
		if alias != "" {
			if cname, ok := exclusions[strings.ToLower(alias)]; ok && c.Name() == cname {
				continue
			}
		}

		ignore, _ := strconv.ParseBool(f.Tag.Get("flagignore"))
		if ignore {
			continue
		}

		// Reject removed flagcustom tag with a migration message.
		if f.Tag.Get("flagcustom") != "" {
			return fmt.Errorf(
				"field %q: flagcustom tag is no longer supported; use RegisterType[T] for per-type hooks or implement FieldHookProvider for per-field hooks",
				f.Name,
			)
		}

		short := f.Tag.Get("flagshort")
		defval := f.Tag.Get("default")
		descr := f.Tag.Get("flagdescr")
		group := f.Tag.Get("flaggroup")
		hidden, _ := strconv.ParseBool(f.Tag.Get("flaghidden"))
		if startingGroup != "" {
			group = startingGroup
		}
		name := internalpath.GetName(path, alias)
		presets, err := internaltag.ParseFlagPresets(f.Tag.Get("flagpreset"))
		if err != nil {
			// Validation should already catch this path. Keep defensive guard for direct/internal callers.
			return fmt.Errorf("field '%s': invalid usage of tag 'flagpreset': %w", path, err)
		}
		if len(presets) > 0 {
			filtered := make([]internaltag.FlagPreset, 0, len(presets))
			for _, preset := range presets {
				if cname, ok := exclusions[strings.ToLower(preset.Name)]; ok && c.Name() == cname {
					continue
				}
				filtered = append(filtered, preset)
			}
			presets = filtered
		}

		// Determine whether to represent hierarchy with the command name.
		// Context-injecting options are treated as shared/common (no command name prefix in env vars).
		// Prefer ContextInjector (standalone); fall back to ContextOptions.
		cName := ""
		_, isContextInjector := o.(ContextInjector)
		_, isContextOptions := o.(ContextOptions)
		if !isContextInjector && !isContextOptions {
			cName = c.Name()
		}

		envs, envMode := internalenv.GetEnv(f, defineEnv, path, alias, cName)
		// Use := to shadow the parameter, matching the original semantics where
		// each field's own tag value drives inheritance for struct recursion
		// but does NOT propagate to subsequent siblings.
		defineEnv := envMode != internalenv.EnvOff
		envOnly := envMode == internalenv.EnvOnly
		mandatory := internaltag.IsMandatory(f) || mandatory

		kind := f.Type.Kind()

		// Lint: suggest flagenv:"only" when flaghidden:"true" + flagenv:"true" is used
		// without any flag-specific tags that would be incompatible with flagenv:"only".
		if hidden && envMode == internalenv.EnvOn && kind != reflect.Struct {
			flagType := f.Tag.Get("flagtype")
			fh, hasFieldHook := fieldHooks[f.Name]
			hasDefineHook := hasFieldHook && fh.Define != nil
			if short == "" && len(presets) == 0 && flagType == "" && !hasDefineHook {
				fmt.Fprintf(c.ErrOrStderr(),
					"structcli: field '%s': flaghidden:\"true\" + flagenv:\"true\" can be replaced with flagenv:\"only\" (which also rejects CLI input)\n",
					f.Name,
				)
			}
		}
		// Lint: flaghidden:"true" is redundant with flagenv:"only" since env-only
		// already forces the flag hidden.
		if hidden && envOnly {
			fmt.Fprintf(c.ErrOrStderr(),
				"structcli: field '%s': flaghidden:\"true\" is redundant with flagenv:\"only\" (env-only fields are always hidden)\n",
				f.Name,
			)
		}
		applyFieldMetadata := func() {
			fs := c.Flags()

			// Persist path metadata on each defined flag so Unmarshal can rebuild
			// remapping state from the current command context (without package globals).
			mustSetAnnotation(fs, name, flagPathAnnotation, []string{path})

			// Marking the flag
			if mandatory {
				mustMarkRequired(c, name)
			}
			if hidden {
				mustMarkHidden(fs, name)
			}

			// Set the defaults
			if defval != "" {
				GetViper(c).SetDefault(name, defval)
				GetViper(c).SetDefault(path, defval)
				// This is needed for the usage help messages
				fs.Lookup(name).DefValue = defval
				mustSetAnnotation(fs, name, flagDefaultAnnotation, []string{defval})
			}

			if len(envs) > 0 {
				mustSetAnnotation(fs, name, internalenv.FlagAnnotation, envs)
			}

			// Set the group annotation on the current flag
			if group != "" {
				mustSetAnnotation(fs, name, internalusage.FlagGroupAnnotation, []string{group})
			}

			// Store enum values as a machine-readable annotation for downstream consumers.
			// Prefer EnumValuer interface (authoritative, type-level) over description parsing (fragile).
			if fl := fs.Lookup(name); fl != nil {
				var enumVals []string
				if ev, ok := fl.Value.(EnumValuer); ok {
					enumVals = ev.EnumValues()
				} else if matches := enumPattern.FindStringSubmatch(fl.Usage); len(matches) > 1 {
					// Fallback: parse {val1,val2,...} from the description for non-EnumValuer flags
					vals := strings.Split(matches[1], ",")
					for i := range vals {
						vals[i] = strings.TrimSpace(vals[i])
					}
					enumVals = vals
				}
				if len(enumVals) > 0 {
					mustSetAnnotation(fs, name, flagEnumAnnotation, enumVals)
				}
			}

			// Store validation struct tag so downstream consumers can inspect rules
			if validateTag := f.Tag.Get(validateTagName); validateTag != "" {
				mustSetAnnotation(fs, name, flagValidateAnnotation, []string{validateTag})
			}

			// Store transformation struct tag so downstream consumers can inspect rules
			if modTag := f.Tag.Get(modTagName); modTag != "" {
				mustSetAnnotation(fs, name, flagModAnnotation, []string{modTag})
			}
		}
		applyPresetAliases := func() {
			fs := c.Flags()
			for _, preset := range presets {
				aliasName := preset.Name
				aliasValue := preset.Value
				targetFlagName := name

				// Avoid redefining when the same options are attached multiple times.
				if fs.Lookup(aliasName) != nil {
					continue
				}

				usage := fmt.Sprintf("alias for --%s=%s", targetFlagName, aliasValue)
				fs.BoolFunc(aliasName, usage, func(raw string) error {
					enabled, err := strconv.ParseBool(raw)
					if err != nil {
						return fmt.Errorf("invalid boolean value for alias --%s: %w", aliasName, err)
					}
					if !enabled {
						return nil
					}

					if err := fs.Set(targetFlagName, aliasValue); err != nil {
						return fmt.Errorf("couldn't apply alias --%s to --%s: %w", aliasName, targetFlagName, err)
					}

					return nil
				})

				if group != "" {
					mustSetAnnotation(fs, aliasName, internalusage.FlagGroupAnnotation, []string{group})
				}
				if hidden {
					mustMarkHidden(fs, aliasName)
				}
			}

			// Store preset metadata on the target flag for machine-readable discovery
			if len(presets) > 0 {
				presetData := make([]string, 0, len(presets))
				for _, preset := range presets {
					presetData = append(presetData, preset.Name+"="+preset.Value)
				}
				mustSetAnnotation(fs, name, flagPresetsAnnotation, presetData)
			}
		}
		finalizeFieldDefinition := func() {
			applyFieldMetadata()
			// Env-only: force hidden and set the env-only annotation.
			// The flag was created normally (correct type, default, etc.)
			// but is now hidden from CLI help and marked for schema/generators.
			if envOnly {
				fs := c.Flags()
				mustMarkHidden(fs, name)
				mustSetAnnotation(fs, name, internalenv.FlagEnvOnlyAnnotation, []string{"true"})
			}
			applyPresetAliases()
			// Register per-field completion hook from FieldCompleter interface.
			// Skipped for envOnly fields: hidden flags have no CLI completion.
			if !envOnly {
				if completeHook, ok := completionHooks[f.Name]; ok {
					if _, exists := c.GetFlagCompletionFunc(name); !exists {
						internalhooks.StoreCompletionHookFuncDirect(c, name, completeHook)
					}
				}
			}
			// Auto-register enum completion when no explicit completion hook exists.
			if _, exists := c.GetFlagCompletionFunc(name); !exists {
				if fl := c.Flags().Lookup(name); fl != nil {
					if ev, ok := fl.Value.(EnumValuer); ok {
						vals := ev.EnumValues()
						if err := c.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
							return vals, cobra.ShellCompDirectiveNoFileComp
						}); err != nil {
							panic(fmt.Sprintf("structcli: RegisterFlagCompletionFunc(%q): %v", name, err))
						}
					}
				}
			}
		}

		// Per-field hooks from FieldHookProvider take highest precedence.
		if fh, ok := fieldHooks[f.Name]; ok && kind != reflect.Struct {
			if fh.Define != nil {
				returnedValue, returnedUsage := fh.Define(name, short, descr, f, field)
				c.Flags().VarP(returnedValue, name, short, returnedUsage)

				if fh.Decode != nil {
					internalhooks.StoreDecodeHookFuncDirect(c, name, fh.Decode, f.Type)
				}

				finalizeFieldDefinition()

				continue
			}
		}

		// Check registry for known custom types (RegisterType, RegisterEnum, built-ins).
		if internalhooks.InferDefineHooks(c, name, short, descr, f, field) {
			if !internalhooks.InferDecodeHooks(c, name, f.Type) {
				return fmt.Errorf("internal error: missing decode hook for built-in type %s", f.Type)
			}

			finalizeFieldDefinition()

			continue
		}

		// Non-standard types without registry hooks or per-field hooks are skipped.
		if !internaltag.IsStandardType(f.Type) && kind != reflect.Struct && kind != reflect.Slice {
			continue
		}

		if c.Flags().Lookup(name) != nil {
			finalizeFieldDefinition()

			continue
		}

		// Standard Go types handled inline via cobra/pflag primitives.
		switch kind {
		case reflect.Struct:
			// NOTE > field.Interface() doesn't work because it actually returns a copy of the object wrapping the interface
			if err := define(c, field.Addr().Interface(), group, path, exclusions, defineEnv, mandatory, validateTagName, modTagName); err != nil {
				return err
			}

			continue

		case reflect.Bool:
			val := field.Interface().(bool)
			ref := field.Addr().Interface().(*bool)
			c.Flags().BoolVarP(ref, name, short, val, descr)

		case reflect.String:
			val := field.Interface().(string)
			ref := field.Addr().Interface().(*string)
			c.Flags().StringVarP(ref, name, short, val, descr)

		case reflect.Uint:
			val := field.Interface().(uint)
			ref := field.Addr().Interface().(*uint)
			c.Flags().UintVarP(ref, name, short, val, descr)

		case reflect.Uint8:
			val := field.Interface().(uint8)
			ref := field.Addr().Interface().(*uint8)
			c.Flags().Uint8VarP(ref, name, short, val, descr)

		case reflect.Uint16:
			val := field.Interface().(uint16)
			ref := field.Addr().Interface().(*uint16)
			c.Flags().Uint16VarP(ref, name, short, val, descr)

		case reflect.Uint32:
			val := field.Interface().(uint32)
			ref := field.Addr().Interface().(*uint32)
			c.Flags().Uint32VarP(ref, name, short, val, descr)

		case reflect.Uint64:
			val := field.Interface().(uint64)
			ref := field.Addr().Interface().(*uint64)
			c.Flags().Uint64VarP(ref, name, short, val, descr)

		case reflect.Int:
			val := field.Interface().(int)
			ref := field.Addr().Interface().(*int)
			if f.Tag.Get("flagtype") == "count" {
				c.Flags().CountVarP(ref, name, short, descr)

				finalizeFieldDefinition()

				continue
			}
			c.Flags().IntVarP(ref, name, short, val, descr)

		case reflect.Int8:
			val := field.Interface().(int8)
			ref := field.Addr().Interface().(*int8)
			c.Flags().Int8VarP(ref, name, short, val, descr)

		case reflect.Int16:
			val := field.Interface().(int16)
			ref := field.Addr().Interface().(*int16)
			c.Flags().Int16VarP(ref, name, short, val, descr)

		case reflect.Int32:
			val := field.Interface().(int32)
			ref := field.Addr().Interface().(*int32)
			c.Flags().Int32VarP(ref, name, short, val, descr)

		case reflect.Int64:
			val := field.Interface().(int64)
			ref := field.Addr().Interface().(*int64)
			c.Flags().Int64VarP(ref, name, short, val, descr)

		case reflect.Float32:
			val := field.Interface().(float32)
			ref := field.Addr().Interface().(*float32)
			c.Flags().Float32VarP(ref, name, short, val, descr)

		case reflect.Float64:
			val := field.Interface().(float64)
			ref := field.Addr().Interface().(*float64)
			c.Flags().Float64VarP(ref, name, short, val, descr)

		case reflect.Slice:
			switch f.Type.Elem().Kind() {
			case reflect.String:
				val := field.Interface().([]string)
				ref := field.Addr().Interface().(*[]string)
				c.Flags().StringSliceVarP(ref, name, short, val, descr)
			case reflect.Int:
				val := field.Interface().([]int)
				ref := field.Addr().Interface().(*[]int)
				c.Flags().IntSliceVarP(ref, name, short, val, descr)
			}
			if !internalhooks.InferDecodeHooks(c, name, f.Type) {
				return fmt.Errorf("internal error: missing decode hook for built-in type %s", f.Type)
			}

		default:
			continue
		}

		finalizeFieldDefinition()
	}

	return nil
}

func Reset() {
	SetEnvPrefix("")
	hookStore.Range(func(key, _ any) bool {
		hookStore.Delete(key)

		return true
	})
	configOnceStore.Range(func(key, _ any) bool {
		configOnceStore.Delete(key)

		return true
	})
}
