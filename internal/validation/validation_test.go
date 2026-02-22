package internalvalidation

import (
	"errors"
	"reflect"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validationCustomType string

type invalidShorthandOpts struct {
	Name string `flagshort:"ab"`
}

type invalidBoolTagOpts struct {
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

type customMissingDefineOpts struct {
	Mode validationCustomType `flagcustom:"true"`
}

type customMissingDecodeOpts struct {
	Mode validationCustomType `flagcustom:"true"`
}

func (o *customMissingDecodeOpts) DefineMode(name, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

type customInvalidDefineSigOpts struct {
	Mode validationCustomType `flagcustom:"true"`
}

func (o *customInvalidDefineSigOpts) DefineMode(_ int, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

func (o *customInvalidDefineSigOpts) DecodeMode(input any) (any, error) {
	return input, nil
}

type customInvalidDecodeSigOpts struct {
	Mode validationCustomType `flagcustom:"true"`
}

func (o *customInvalidDecodeSigOpts) DefineMode(name, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

func (o *customInvalidDecodeSigOpts) DecodeMode(_ string) string {
	return ""
}

type customValidOpts struct {
	Mode validationCustomType `flagcustom:"true"`
}

func (o *customValidOpts) DefineMode(name, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

func (o *customValidOpts) DecodeMode(input any) (any, error) {
	return input, nil
}

type customConflictingTypeOpts struct {
	Mode1 validationCustomType `flagcustom:"true"`
	Mode2 validationCustomType `flagcustom:"true"`
}

func (o *customConflictingTypeOpts) DefineMode1(name, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

func (o *customConflictingTypeOpts) DecodeMode1(input any) (any, error) {
	return input, nil
}

func (o *customConflictingTypeOpts) DefineMode2(name, short, descr string, _ reflect.StructField, _ reflect.Value) (pflag.Value, string) {
	return nil, descr
}

func (o *customConflictingTypeOpts) DecodeMode2(input any) (any, error) {
	return input, nil
}

type invalidCompleteSigOpts struct {
	Mode string `flag:"mode"`
}

func (o *invalidCompleteSigOpts) CompleteMode(_ string) []string {
	return []string{"dev"}
}

type validCompleteSigOpts struct {
	Mode string `flag:"mode"`
}

func (o *validCompleteSigOpts) CompleteMode(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"dev", "prod"}, cobra.ShellCompDirectiveNoFileComp
}

type ignoredInvalidCompleteSigOpts struct {
	Hidden string `flag:"hidden" flagignore:"true"`
}

func (o *ignoredInvalidCompleteSigOpts) CompleteHidden(_ string) []string {
	return []string{"ignored"}
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
		{name: "invalid bool tag", opts: &invalidBoolTagOpts{}, err: structclierrors.ErrInvalidBooleanTag},
		{name: "conflicting tags", opts: &conflictingTagsOpts{}, err: structclierrors.ErrConflictingTags},
		{name: "invalid flag name", opts: &invalidFlagNameOpts{}, err: structclierrors.ErrInvalidFlagName},
		{name: "duplicate flag", opts: &duplicateFlagOpts{}, err: structclierrors.ErrDuplicateFlag},
		{name: "invalid preset syntax", opts: &invalidPresetSyntaxOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset on struct", opts: &presetOnStructOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset with ignore", opts: &presetWithIgnoreOpts{}, err: structclierrors.ErrInvalidTagUsage},
		{name: "preset duplicate with flag", opts: &presetDuplicateWithFlagOpts{}, err: structclierrors.ErrDuplicateFlag},
		{name: "preset duplicate across fields", opts: &presetDuplicateAcrossFieldsOpts{}, err: structclierrors.ErrDuplicateFlag},
		{name: "missing define hook", opts: &customMissingDefineOpts{}, err: structclierrors.ErrMissingDefineHook},
		{name: "missing decode hook", opts: &customMissingDecodeOpts{}, err: structclierrors.ErrMissingDecodeHook},
		{name: "invalid define signature", opts: &customInvalidDefineSigOpts{}, err: structclierrors.ErrInvalidDefineHookSignature},
		{name: "invalid decode signature", opts: &customInvalidDecodeSigOpts{}, err: structclierrors.ErrInvalidDecodeHookSignature},
		{name: "invalid completion signature", opts: &invalidCompleteSigOpts{}, err: structclierrors.ErrInvalidCompleteHookSignature},
		{name: "conflicting custom type", opts: &customConflictingTypeOpts{}, err: structclierrors.ErrConflictingType},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := Struct(cmd, tt.opts)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.err), err.Error())
		})
	}
}

func TestStructValidationSuccess(t *testing.T) {
	require.NoError(t, Struct(&cobra.Command{Use: "app-1"}, &customValidOpts{}))
	require.NoError(t, Struct(&cobra.Command{Use: "app-2"}, &validCompleteSigOpts{}))
	require.NoError(t, Struct(&cobra.Command{Use: "app-3"}, &ignoredInvalidCompleteSigOpts{}))
}

func TestStructValidationNilInput(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	err := Struct(cmd, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, structclierrors.ErrInputValue))
}
