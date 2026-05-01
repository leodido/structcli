package internalvalidation

import (
	"errors"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invalidShorthandOpts struct {
	Name string `flagshort:"ab"`
}

type invalidFlagEnvTagOpts struct {
	Name string `flagenv:"oops"`
}

type conflictingTagsOpts struct {
	Name string `flagrequired:"true" flagignore:"true"`
}

type invalidFlagNameOpts struct {
	Name string `flag:"bad name"`
}

type duplicateFlagOpts struct {
	A string `flag:"same"`
	B string `flag:"same"`
}

type invalidPresetSyntaxOpts struct {
	Level int `flagpreset:"logeverything"`
}

type presetOnStructOpts struct {
	Nested struct {
		Level int
	} `flagpreset:"logeverything=5"`
}

type presetWithIgnoreOpts struct {
	Level int `flagignore:"true" flagpreset:"logeverything=5"`
}

type presetDuplicateWithFlagOpts struct {
	Level int `flag:"logeverything" flagpreset:"logeverything=5"`
}

type presetDuplicateAcrossFieldsOpts struct {
	A int `flagpreset:"logeverything=5"`
	B int `flagpreset:"logeverything=4"`
}

func TestIsValidBoolTag(t *testing.T) {
	val, err := IsValidBoolTag("X", "flagenv", "")
	require.NoError(t, err)
	assert.Nil(t, val)

	val, err = IsValidBoolTag("X", "flagenv", "true")
	require.NoError(t, err)
	require.NotNil(t, val)
	assert.True(t, *val)

	_, err = IsValidBoolTag("X", "flagenv", "not-bool")
	require.Error(t, err)
	assert.True(t, errors.Is(err, structclierrors.ErrInvalidBooleanTag))
}

func TestStructValidationErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}

	cases := []struct {
		name string
		opts any
		err  error
	}{
		{name: "invalid shorthand", opts: &invalidShorthandOpts{}, err: structclierrors.ErrInvalidShorthand},
		{name: "invalid flagenv tag", opts: &invalidFlagEnvTagOpts{}, err: structclierrors.ErrInvalidFlagEnvTag},
		{name: "conflicting tags", opts: &conflictingTagsOpts{}, err: structclierrors.ErrConflictingTags},
		{name: "invalid flag name", opts: &invalidFlagNameOpts{}, err: structclierrors.ErrInvalidFlagName},
		{name: "duplicate flag", opts: &duplicateFlagOpts{}, err: structclierrors.ErrDuplicateFlag},
		{name: "invalid preset syntax", opts: &invalidPresetSyntaxOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset on struct", opts: &presetOnStructOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset with ignore", opts: &presetWithIgnoreOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset duplicate with flag", opts: &presetDuplicateWithFlagOpts{}, err: structclierrors.ErrDuplicateFlag},
		{name: "preset duplicate across fields", opts: &presetDuplicateAcrossFieldsOpts{}, err: structclierrors.ErrDuplicateFlag},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := Struct(cmd, tt.opts)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.err), err.Error())
		})
	}
}

type simpleValidOpts struct {
	Name string `flag:"name"`
}

func TestStructValidationSuccess(t *testing.T) {
	require.NoError(t, Struct(&cobra.Command{Use: "app-1"}, &simpleValidOpts{}))
}

func TestStructValidationNilInput(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	err := Struct(cmd, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, structclierrors.ErrInputValue))
}
