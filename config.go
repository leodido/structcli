package structcli

import (
	"fmt"
	"sync"

	"github.com/leodido/structcli/config"
	internalconfig "github.com/leodido/structcli/internal/config"
	internalenv "github.com/leodido/structcli/internal/env"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	defaultSearchPaths = []config.SearchPathType{
		config.SearchPathEtc,
		config.SearchPathExecutableDirHidden,
		config.SearchPathHomeHidden,
		config.SearchPathWorkingDirHidden,
	}

	configRootMu sync.RWMutex
	configRoot   *cobra.Command
)

const configValidateKeysAnnotation = "___leodido_structcli_config_validate_keys"

func setConfigRoot(rootC *cobra.Command) {
	configRootMu.Lock()
	defer configRootMu.Unlock()
	configRoot = rootC
}

func getConfigRoot() *cobra.Command {
	configRootMu.RLock()
	defer configRootMu.RUnlock()

	return configRoot
}

func clearConfigRoot(rootC *cobra.Command) {
	configRootMu.Lock()
	defer configRootMu.Unlock()
	if configRoot == rootC {
		configRoot = nil
	}
}

func validateConfigKeysEnabled(c *cobra.Command) bool {
	if c == nil {
		return false
	}
	rootC := c.Root()
	if rootC == nil || rootC.Annotations == nil {
		return false
	}

	return rootC.Annotations[configValidateKeysAnnotation] == "true"
}

// SetupConfig creates the --config global flag and wires config discovery for the root command.
//
// Works only for the root command.
//
// Call this before attaching/defining options when you rely on app-prefixed
// environment variables (eg. FULL_*), because SetupConfig is what initializes
// the global env prefix used while defining env annotations.
//
// Configuration file data is loaded into a root-scoped config viper
// (see GetConfigViper), then merged into the active command scoped viper
// during UseConfig/Unmarshal.
//
// Set config.Options.ValidateKeys to enable strict config-key validation
// during Unmarshal for command-relevant config entries.
func SetupConfig(rootC *cobra.Command, cfgOpts config.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupConfig must be called on the root command")
	}

	// Determine the app name
	appName := GetOrSetAppName(cfgOpts.AppName, rootC.Name())
	if appName == "" {
		return fmt.Errorf("couldn't determine the app name")
	}

	// Apply defaults
	if cfgOpts.FlagName == "" {
		cfgOpts.FlagName = "config"
	}
	if cfgOpts.ConfigName == "" {
		cfgOpts.ConfigName = "config"
	}
	if cfgOpts.EnvVar == "" {
		if cfgOpts.AppName == "" {
			if currentPrefix := EnvPrefix(); currentPrefix != "" {
				cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", currentPrefix)
			}
		} else {
			cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", internalenv.NormEnv(appName))
		}
	} else {
		cfgOpts.EnvVar = internalenv.NormEnv(cfgOpts.EnvVar)
	}
	if len(cfgOpts.SearchPaths) == 0 {
		cfgOpts.SearchPaths = defaultSearchPaths
	}

	configFile := ""

	// Add persistent flag to root command
	rootC.PersistentFlags().StringVar(&configFile, cfgOpts.FlagName, configFile, internalconfig.Description(appName, cfgOpts))

	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	if cfgOpts.ValidateKeys {
		rootC.Annotations[configValidateKeysAnnotation] = "true"
	} else {
		delete(rootC.Annotations, configValidateKeysAnnotation)
	}

	// Add filename completion
	extensions := []string{"yaml", "yml", "json", "toml"}
	if err := rootC.MarkPersistentFlagFilename(cfgOpts.FlagName, extensions...); err != nil {
		return fmt.Errorf("couldn't set filename completion: %w", err)
	}

	// Set up viper configuration
	setConfigRoot(rootC)
	cobra.OnInitialize(func() {
		configVip := internalscope.Get(rootC).ConfigViper()
		internalconfig.SetupConfig(configVip, configFile, appName, cfgOpts)
	})

	// Store cleanup function
	cobra.OnFinalize(func() {
		internalscope.Get(rootC).ResetConfigViper()
		clearConfigRoot(rootC)
		viper.Reset()
	})

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

	return nil
}

// UseConfig attempts to read the configuration file based on the provided condition.
//
// The readWhen function determines whether config reading should be attempted.
// Returns whether config was loaded, a status message, and any error encountered.
//
// When SetupConfig was configured, this reads into the root-scoped config viper
// and merges command-relevant settings into the active command scoped viper.
//
// If SetupConfig was not called, UseConfig falls back to reading on the global
// viper singleton. Prefer SetupConfig for deterministic command-scoped behavior.
func UseConfig(readWhen func() bool) (inUse bool, mes string, err error) {
	if rootC := getConfigRoot(); rootC != nil {
		return useConfigForCommand(rootC, readWhen)
	}

	// Fallback for callers that use UseConfig without SetupConfig.
	return useConfigOnViper(viper.GetViper(), readWhen)
}

func useConfigForCommand(c *cobra.Command, readWhen func() bool) (inUse bool, mes string, err error) {
	if c == nil {
		return useConfigOnViper(viper.GetViper(), readWhen)
	}

	rootVip := internalscope.Get(c.Root()).ConfigViper()
	inUse, mes, err = useConfigOnViper(rootVip, readWhen)
	if err != nil || !inUse {
		return inUse, mes, err
	}

	configToMerge := internalconfig.Merge(rootVip.AllSettings(), c)
	if err := internalscope.Get(c).Viper().MergeConfigMap(configToMerge); err != nil {
		return false, "", fmt.Errorf("error merging config for command %q: %w", c.CommandPath(), err)
	}

	return inUse, mes, nil
}

func useConfigOnViper(vip *viper.Viper, readWhen func() bool) (inUse bool, mes string, err error) {
	// Use the readWhen function to determine if we should read config
	if readWhen != nil && !readWhen() {
		return false, "", nil
	}

	if err := vip.ReadInConfig(); err == nil {
		return true, fmt.Sprintf("Using config file: %s", vip.ConfigFileUsed()), nil
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, ignore...
			return false, "Running without a configuration file", nil
		} else {
			// Config file was found but another error was produced
			return false, "", fmt.Errorf("error running with config file: %s: %w", vip.ConfigFileUsed(), err)
		}
	}
}

// UseConfigSimple is a simpler version of UseConfig that uses c.IsAvailableCommand() as the readWhen function.
//
// It does not check for the config file when the command is not available (eg., help).
//
// The config file (if found) is loaded through the root-scoped config viper and
// merged into c's effective scoped viper.
func UseConfigSimple(c *cobra.Command) (inUse bool, message string, err error) {
	return useConfigForCommand(c, func() bool {
		return c.IsAvailableCommand()
	})
}
