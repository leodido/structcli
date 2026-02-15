package internalconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leodido/structcli/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestSetupConfig_ExplicitFileHasPrecedence(t *testing.T) {
	vip := viper.New()

	tmpDir := t.TempDir()
	explicitFile := filepath.Join(tmpDir, "explicit.yaml")
	envFile := filepath.Join(tmpDir, "env.yaml")

	writeConfigFile(t, explicitFile, "source: explicit\n")
	writeConfigFile(t, envFile, "source: env\n")

	t.Setenv("TESTAPP_CONFIG", envFile)

	opts := config.Options{
		EnvVar:      "TESTAPP_CONFIG",
		ConfigName:  "config",
		SearchPaths: []config.SearchPathType{config.SearchPathCustom},
		CustomPaths: []string{tmpDir},
	}

	SetupConfig(vip, explicitFile, "testapp", opts)
	require.NoError(t, vip.ReadInConfig())

	assert.Equal(t, explicitFile, vip.ConfigFileUsed())
	assert.Equal(t, "explicit", vip.GetString("source"))
}

func TestSetupConfig_EnvFileHasPrecedenceOverSearchPaths(t *testing.T) {
	vip := viper.New()

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env.yaml")
	searchFile := filepath.Join(tmpDir, "search.yaml")

	writeConfigFile(t, envFile, "source: env\n")
	writeConfigFile(t, searchFile, "source: search\n")

	t.Setenv("TESTAPP_CONFIG", envFile)

	opts := config.Options{
		EnvVar:      "TESTAPP_CONFIG",
		ConfigName:  "search",
		SearchPaths: []config.SearchPathType{config.SearchPathCustom},
		CustomPaths: []string{tmpDir},
	}

	SetupConfig(vip, "", "testapp", opts)
	require.NoError(t, vip.ReadInConfig())

	assert.Equal(t, envFile, vip.ConfigFileUsed())
	assert.Equal(t, "env", vip.GetString("source"))
}

func TestSetupConfig_SearchPathFallbackIsUsedWhenNoExplicitOrEnv(t *testing.T) {
	vip := viper.New()

	tmpDir := t.TempDir()
	appName := "testapp"
	resolvedPath := filepath.Join(tmpDir, appName)
	configFile := filepath.Join(resolvedPath, "settings.yaml")
	writeConfigFile(t, configFile, "source: search\n")

	t.Setenv("TESTAPP_CONFIG", "")

	opts := config.Options{
		EnvVar:      "TESTAPP_CONFIG",
		ConfigName:  "settings",
		SearchPaths: []config.SearchPathType{config.SearchPathCustom},
		CustomPaths: []string{filepath.Join(tmpDir, "{APP}")},
	}

	SetupConfig(vip, "", appName, opts)
	require.NoError(t, vip.ReadInConfig())

	assert.Equal(t, configFile, vip.ConfigFileUsed())
	assert.Equal(t, "search", vip.GetString("source"))
}

func TestResolveSearchPaths_CustomPathsAddedOnce(t *testing.T) {
	paths := resolveSearchPaths(
		[]config.SearchPathType{
			config.SearchPathCustom,
			config.SearchPathEtc,
			config.SearchPathCustom,
		},
		[]string{
			"/a/{APP}",
			"/b/{APP}",
		},
		"myapp",
		false,
	)

	assert.Equal(t, []string{"/a/myapp", "/b/myapp", "/etc/myapp"}, paths)
}

func TestResolveSearchPath_ExpandsEnvPwdAndAppPlaceholder(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	t.Setenv("ROOT", "/tmp/base")
	// Keep a literal $PWD after ExpandEnv, so resolveSearchPath executes its
	// explicit $PWD handling branch.
	t.Setenv("PWD", "$PWD")

	out := resolveSearchPath("${ROOT}/{APP}/$PWD", "myapp")

	assert.NotContains(t, out, "{APP}")
	assert.NotContains(t, out, "$PWD")
	assert.Contains(t, out, "/tmp/base/myapp")
	assert.Contains(t, out, wd)
}

func TestDescription_NoSearchPaths(t *testing.T) {
	desc := Description("myapp", config.Options{ConfigName: "cfg", SearchPaths: nil})
	assert.Equal(t, "config file", desc)
}

func TestDescription_LimitsToFirstThreePaths(t *testing.T) {
	opts := config.Options{
		ConfigName: "cfg",
		SearchPaths: []config.SearchPathType{
			config.SearchPathCustom,
			config.SearchPathEtc,
			config.SearchPathHomeHidden,
			config.SearchPathWorkingDirHidden,
		},
		CustomPaths: []string{"/x/{APP}"},
	}

	templatePaths := resolveSearchPaths(opts.SearchPaths, opts.CustomPaths, "myapp", true)
	require.Greater(t, len(templatePaths), 3)

	expected := fmt.Sprintf(
		"config file (fallbacks to: {%s}/%s.{yaml,json,toml})",
		strings.Join(templatePaths[:3], ","),
		opts.ConfigName,
	)

	assert.Equal(t, expected, Description("myapp", opts))
}
