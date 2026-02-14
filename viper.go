package structcli

import (
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	structclierrors "github.com/leodido/structcli/errors"
	internalconfig "github.com/leodido/structcli/internal/config"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	flagPathAnnotation    = "___leodido_structcli_flagpath"
	flagDefaultAnnotation = "___leodido_structcli_flagdefault"
)

func remappingMetadataFromCommand(c *cobra.Command) (map[string]string, map[string]string) {
	aliasToPathMap := make(map[string]string)
	defaultsMap := make(map[string]string)
	seen := make(map[string]struct{})

	for comm := c; comm != nil; comm = comm.Parent() {
		comm.LocalFlags().VisitAll(func(f *pflag.Flag) {
			// Prefer nearest command definition when duplicated along ancestry.
			if _, ok := seen[f.Name]; ok {
				return
			}
			seen[f.Name] = struct{}{}

			if pathMetadata, ok := f.Annotations[flagPathAnnotation]; ok && len(pathMetadata) > 0 {
				path := pathMetadata[0]
				if path != "" && path != f.Name {
					aliasToPathMap[f.Name] = path
				}
			}

			if defaultMetadata, ok := f.Annotations[flagDefaultAnnotation]; ok && len(defaultMetadata) > 0 {
				defaultsMap[f.Name] = defaultMetadata[0]
			}
		})
	}

	return aliasToPathMap, defaultsMap
}

// GetViper returns the viper instance associated with the given command.
//
// Each command has its own isolated viper instance for configuration management.
func GetViper(c *cobra.Command) *viper.Viper {
	s := internalscope.Get(c)

	return s.Viper()
}

// Unmarshal populates the options struct with values from flags, environment variables,
// and configuration files.
//
// It automatically handles decode hooks, validation, transformation, and context updates based on the options type.
func Unmarshal(c *cobra.Command, opts Options, hooks ...mapstructure.DecodeHookFunc) error {
	scope := internalscope.Get(c)
	vip := scope.Viper()

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := internalconfig.Merge(viper.AllSettings(), c)
	vip.MergeConfigMap(configToMerge)

	aliasToPathMap, defaultsMap := remappingMetadataFromCommand(c)

	// Re-apply explicit struct tag defaults to the command-scoped viper.
	// Defaults are initially set during Define on that command's scope; when Unmarshal
	// is executed on a leaf command, we must set them again on the leaf scope.
	for name, defval := range defaultsMap {
		vip.SetDefault(name, defval)
		if path, ok := aliasToPathMap[name]; ok {
			vip.SetDefault(path, defval)
		}
	}

	// Use `KeyRemappingHook` for smart config keys
	hooks = append([]mapstructure.DecodeHookFunc{internalconfig.KeyRemappingHook(aliasToPathMap, defaultsMap)}, hooks...)

	// Look for decode hook annotation appending them to the list of hooks to use for unmarshalling
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if decodeHooks, defineDecodeHooks := f.Annotations[internalhooks.FlagDecodeHookAnnotation]; defineDecodeHooks {
			for _, decodeHook := range decodeHooks {
				// Custom decode hook have precedence
				if customDecodeHook, customDecodeHookExists := scope.GetCustomDecodeHook(decodeHook); customDecodeHookExists {
					hooks = append(hooks, customDecodeHook)

					continue
				}

				// Check the registry for built-in decode hooks
				if decodeHookFunc, ok := internalhooks.AnnotationToDecodeHookRegistry[decodeHook]; ok {
					hooks = append(hooks, decodeHookFunc)
				}
			}
		}
	})

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		hooks...,
	))

	if err := vip.Unmarshal(opts /*custonNameHook,*/, decodeHook); err != nil {
		return fmt.Errorf("couldn't unmarshal config to options: %w", err)
	}

	// Automatically set common options into the context of the cobra command
	if o, ok := opts.(ContextOptions); ok {
		c.SetContext(o.Context(c.Context()))
	}

	// Automatically transform options if feasible
	if o, ok := opts.(TransformableOptions); ok {
		if transformErr := o.Transform(c.Context()); transformErr != nil {
			return fmt.Errorf("couldn't transform options: %w", transformErr)
		}
	}

	// Automatically run options validation if feasible
	if o, ok := opts.(ValidatableOptions); ok {
		if validationErrors := o.Validate(c.Context()); validationErrors != nil {
			return &structclierrors.ValidationError{
				ContextName: c.Name(),
				Errors:      validationErrors,
			}
		}
	}

	internalconfig.SyncMandatoryFlags(c, reflect.TypeOf(opts), vip, "")

	// Automatic debug output if debug is on
	UseDebug(c, c.OutOrStdout())

	return nil
}
