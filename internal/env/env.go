package internalenv

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	prefix atomic.Value
	EnvSep = "_"
	envRep = strings.NewReplacer("-", EnvSep, ".", EnvSep)
)

const (
	FlagAnnotation        = "leodido/structcli/flag-envs"
	FlagEnvOnlyAnnotation = "leodido/structcli/flag-env-only"
)

func NormEnv(str string) string {
	return envRep.Replace(strings.ToUpper(str))
}

func init() {
	prefix.Store("")
}

func SetPrefix(v string) {
	prefix.Store(v)
}

func GetPrefix() string {
	if current, ok := prefix.Load().(string); ok {
		return current
	}

	return ""
}

// EnvMode describes how a field participates in environment variable binding.
type EnvMode int

const (
	// EnvOff means no env binding for this field.
	EnvOff EnvMode = iota
	// EnvOn means the field has both a CLI flag and env binding.
	EnvOn
	// EnvOnly means the field is settable only via env var (and config), not CLI.
	EnvOnly
)

// IsEnvOnly returns true if the struct field's flagenv tag is set to "only".
func IsEnvOnly(f reflect.StructField) bool {
	return strings.EqualFold(f.Tag.Get("flagenv"), "only")
}

// IsValidFlagEnvTag validates the flagenv tag value.
// Returns nil for valid values ("", "true", "false", "only") and an error otherwise.
func IsValidFlagEnvTag(tagValue string) bool {
	if tagValue == "" {
		return true
	}
	if strings.EqualFold(tagValue, "only") {
		return true
	}
	_, err := strconv.ParseBool(tagValue)

	return err == nil
}

func GetEnv(f reflect.StructField, inherit bool, path, alias, envPrefix string) ([]string, EnvMode) {
	ret := []string{}
	currentPrefix := GetPrefix()

	env := f.Tag.Get("flagenv")
	envOnly := strings.EqualFold(env, "only")
	defineEnv, _ := strconv.ParseBool(env)

	if defineEnv || envOnly || inherit {
		envPath := path
		envAlias := alias

		// Apply env prefix to current env variable
		// But avoid double prefixing if the given prefix matches the global prefix (usually the CLI/app name)
		if envPrefix != "" {
			// Extract app name from prefix (remove trailing underscore and lowercase)
			appName := strings.ToLower(strings.TrimSuffix(currentPrefix, "_"))
			if envPrefix != appName {
				envPath = envPrefix + "." + path
				if alias != "" {
					envAlias = envPrefix + "." + alias
				}
			}
		}

		ret = append(ret, currentPrefix+NormEnv(envPath))
		if alias != "" && path != alias {
			ret = append(ret, currentPrefix+NormEnv(envAlias))
		}
	}

	if envOnly {
		return ret, EnvOnly
	}
	if defineEnv {
		return ret, EnvOn
	}

	return ret, EnvOff
}

// PatchEnvPrefix updates env annotations on all flags of c to use newPrefix.
// It strips any existing oldPrefix from annotation values and prepends newPrefix.
// When oldPrefix is empty and c is the root command, the root command's name was
// used as a pseudo-prefix by GetEnv and must be stripped before applying newPrefix.
// For each patched flag, it clears the bound-env marker in the command's scope
// so that a subsequent BindEnv call will re-bind with the corrected env var names.
func PatchEnvPrefix(c *cobra.Command, oldPrefix, newPrefix string) {
	s := internalscope.Get(c)

	// When no global prefix existed, GetEnv baked the command name into env
	// vars as a pseudo-prefix. For the root command, that pseudo-prefix
	// should be replaced by the real app prefix.
	stripPrefix := oldPrefix
	if oldPrefix == "" && c.Parent() == nil {
		stripPrefix = NormEnv(c.Name()) + EnvSep
	}

	c.Flags().VisitAll(func(f *pflag.Flag) {
		envs, ok := f.Annotations[FlagAnnotation]
		if !ok || len(envs) == 0 {
			return
		}

		patched := make([]string, len(envs))
		for i, env := range envs {
			bare := strings.TrimPrefix(env, stripPrefix)
			patched[i] = newPrefix + bare
		}
		f.Annotations[FlagAnnotation] = patched

		// Clear bound state so BindEnv will re-bind with the new names.
		s.ClearBoundEnv(f.Name)
	})
}

func BindEnv(c *cobra.Command) error {
	s := internalscope.Get(c)
	var bindErr error

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if bindErr != nil {
			return
		}
		if envs, defineEnv := f.Annotations[FlagAnnotation]; defineEnv {
			// Only bind if we haven't already bound this env var for this command
			if !s.IsEnvBound(f.Name) {
				s.SetBound(f.Name)
				input := []string{f.Name}
				input = append(input, envs...)
				if err := s.Viper().BindEnv(input...); err != nil {
					bindErr = fmt.Errorf("couldn't bind env for flag %s: %w", f.Name, err)
				}
			}
		}
	})

	return bindErr
}
