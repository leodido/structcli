package structcli

import (
	"context"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	structclierrors "github.com/leodido/structcli/errors"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flagPresetOptions struct {
	LogLevel int `flag:"loglevel" flagenv:"true" flagpreset:"logeverything=5"`
}

func (o *flagPresetOptions) Attach(c *cobra.Command) error { return nil }

type requiredFlagPresetOptions struct {
	LogLevel int `flag:"loglevel" flagrequired:"true" flagpreset:"logeverything=5"`
}

func (o *requiredFlagPresetOptions) Attach(c *cobra.Command) error { return nil }

type validatedPresetOptions struct {
	LogLevel int `flag:"loglevel" flagpreset:"logeverything=5;logquiet=0;lognormal=3" validate:"min=1,max=4"`
}

func (o *validatedPresetOptions) Attach(c *cobra.Command) error { return nil }

func (o *validatedPresetOptions) Validate(_ context.Context) []error {
	err := validator.New().Struct(o)
	if err == nil {
		return nil
	}
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		errs := make([]error, 0, len(validationErrs))
		for _, fieldErr := range validationErrs {
			errs = append(errs, fieldErr)
		}

		return errs
	}

	return []error{err}
}

type transformThenValidatePresetOptions struct {
	Level string `flag:"level" flagpreset:"logdebug=DEBUG" validate:"oneof=debug info warn"`
}

func (o *transformThenValidatePresetOptions) Attach(c *cobra.Command) error { return nil }

func (o *transformThenValidatePresetOptions) Transform(_ context.Context) error {
	o.Level = strings.ToLower(strings.TrimSpace(o.Level))

	return nil
}

func (o *transformThenValidatePresetOptions) Validate(_ context.Context) []error {
	err := validator.New().Struct(o)
	if err == nil {
		return nil
	}
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		errs := make([]error, 0, len(validationErrs))
		for _, fieldErr := range validationErrs {
			errs = append(errs, fieldErr)
		}

		return errs
	}

	return []error{err}
}

func resetFlagPresetTestState() {
	viper.Reset()
	Reset()
	SetEnvPrefix("")
}

func newFlagPresetCommand(t *testing.T) (*cobra.Command, *flagPresetOptions) {
	t.Helper()

	cmd := &cobra.Command{Use: "app"}
	opts := &flagPresetOptions{}
	require.NoError(t, Define(cmd, opts))

	return cmd, opts
}

func TestDefine_FlagPresetAlias_CreatesFlagWithoutEnvBinding(t *testing.T) {
	resetFlagPresetTestState()

	cmd, _ := newFlagPresetCommand(t)

	aliasFlag := cmd.Flags().Lookup("logeverything")
	require.NotNil(t, aliasFlag)
	assert.Contains(t, aliasFlag.Usage, "alias for --loglevel=5")
	assert.Nil(t, aliasFlag.Annotations[internalenv.FlagAnnotation])
}

func TestUnmarshal_FlagPresetAlias_FlagOrderDefinesWinner(t *testing.T) {
	t.Run("alias_then_canonical", func(t *testing.T) {
		resetFlagPresetTestState()

		cmd, opts := newFlagPresetCommand(t)
		require.NoError(t, cmd.Flags().Parse([]string{"--logeverything", "--loglevel=4"}))
		require.NoError(t, Unmarshal(cmd, opts))

		assert.Equal(t, 4, opts.LogLevel)
	})

	t.Run("canonical_then_alias", func(t *testing.T) {
		resetFlagPresetTestState()

		cmd, opts := newFlagPresetCommand(t)
		require.NoError(t, cmd.Flags().Parse([]string{"--loglevel=4", "--logeverything"}))
		require.NoError(t, Unmarshal(cmd, opts))

		assert.Equal(t, 5, opts.LogLevel)
	})
}

func TestUnmarshal_FlagPresetAlias_HasFlagPrecedenceOverEnvAndConfig(t *testing.T) {
	resetFlagPresetTestState()
	SetEnvPrefix("APP")
	t.Setenv("APP_LOGLEVEL", "3")

	cmd, opts := newFlagPresetCommand(t)
	GetConfigViper(cmd).Set("loglevel", 2)

	require.NoError(t, cmd.Flags().Parse([]string{"--logeverything"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, 5, opts.LogLevel)
}

func TestUnmarshal_FlagPresetAlias_NotAcceptedAsConfigKeyInStrictMode(t *testing.T) {
	resetFlagPresetTestState()

	cmd, opts := newFlagPresetCommand(t)
	cmd.Annotations = map[string]string{
		configValidateKeysAnnotation: "true",
	}
	GetConfigViper(cmd).Set("logeverything", true)

	err := Unmarshal(cmd, opts)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown config keys")
	assert.ErrorContains(t, err, "logeverything")
}

func TestDefine_FlagPresetAlias_SatisfiesRequiredCanonicalFlag(t *testing.T) {
	resetFlagPresetTestState()

	opts := &requiredFlagPresetOptions{}
	cmd := &cobra.Command{
		Use: "app",
		RunE: func(c *cobra.Command, _ []string) error {
			return Unmarshal(c, opts)
		},
	}
	require.NoError(t, Define(cmd, opts))

	cmd.SetArgs([]string{"--logeverything"})
	require.NoError(t, cmd.Execute())
	assert.Equal(t, 5, opts.LogLevel)
}

func TestUnmarshal_FlagPresetAlias_UsesSameValidationPathAsCanonicalFlag(t *testing.T) {
	t.Run("invalid_alias_values_fail_validation", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
			want int
			tag  string
		}{
			{name: "logeverything", args: []string{"--logeverything"}, want: 5, tag: "max"},
			{name: "logquiet", args: []string{"--logquiet"}, want: 0, tag: "min"},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				resetFlagPresetTestState()

				opts := &validatedPresetOptions{}
				cmd := &cobra.Command{Use: "app"}
				require.NoError(t, Define(cmd, opts))
				require.NoError(t, cmd.Flags().Parse(tt.args))

				err := Unmarshal(cmd, opts)
				require.Error(t, err)
				assert.Equal(t, tt.want, opts.LogLevel)

				var valErr *structclierrors.ValidationError
				require.ErrorAs(t, err, &valErr)
				assert.Contains(t, err.Error(), "invalid options for app")
				assert.Contains(t, err.Error(), "failed on the '"+tt.tag+"' tag")
			})
		}
	})

	t.Run("valid_alias_value_passes_validation", func(t *testing.T) {
		resetFlagPresetTestState()

		opts := &validatedPresetOptions{}
		cmd := &cobra.Command{Use: "app"}
		require.NoError(t, Define(cmd, opts))
		require.NoError(t, cmd.Flags().Parse([]string{"--lognormal"}))

		require.NoError(t, Unmarshal(cmd, opts))
		assert.Equal(t, 3, opts.LogLevel)
	})
}

func TestUnmarshal_FlagPresetAlias_GoesThroughTransformThenValidate(t *testing.T) {
	resetFlagPresetTestState()

	opts := &transformThenValidatePresetOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--logdebug"}))

	require.NoError(t, Unmarshal(cmd, opts))
	assert.Equal(t, "debug", opts.Level)
}
