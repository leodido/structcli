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
	FlagAnnotation = "___leodido_structcli_flagenvs"
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

func GetEnv(f reflect.StructField, inherit bool, path, alias, envPrefix string) ([]string, bool) {
	ret := []string{}
	currentPrefix := GetPrefix()

	env := f.Tag.Get("flagenv")
	defineEnv, _ := strconv.ParseBool(env)

	if defineEnv || inherit {
		if f.Type.Kind() != reflect.Struct {
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
	}

	return ret, defineEnv
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
