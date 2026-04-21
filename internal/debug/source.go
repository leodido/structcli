package internaldebug

import (
	"os"

	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FlagSource identifies where a flag's resolved value came from.
type FlagSource string

const (
	SourceFlag    FlagSource = "flag"
	SourceEnv     FlagSource = "env"
	SourceConfig  FlagSource = "config"
	SourceDefault FlagSource = "default"
)

// ResolveFlagSource determines where a flag's value came from.
//
// Priority: flag (explicitly set on CLI) > env > config > default.
func ResolveFlagSource(f *pflag.Flag, configViper *viper.Viper) FlagSource {
	if f.Changed {
		return SourceFlag
	}

	// Check if a bound env var is present in the environment.
	// This checks presence (os.LookupEnv), not whether viper actually used it,
	// which is correct given structcli's priority order: when an env var is set
	// and the flag wasn't explicitly passed, the env var is the effective source.
	if envs, ok := f.Annotations[internalenv.FlagAnnotation]; ok {
		for _, envVar := range envs {
			if _, set := os.LookupEnv(envVar); set {
				return SourceEnv
			}
		}
	}

	// Check if the config viper has this key.
	if configViper != nil && configViper.IsSet(f.Name) {
		return SourceConfig
	}

	return SourceDefault
}
