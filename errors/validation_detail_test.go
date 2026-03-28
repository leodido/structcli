package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStruct is used to trigger real validator.FieldError instances.
type testStruct struct {
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18"`
	Name  string `validate:"required"`
}

func TestDetails_ExtractsValidatorFieldErrors(t *testing.T) {
	v := validator.New()
	err := v.Struct(&testStruct{
		Email: "not-an-email",
		Age:   10,
		Name:  "Alice",
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	// Build a ValidationError the same way structcli does
	errs := make([]error, len(valErrs))
	for i, fe := range valErrs {
		errs[i] = fe
	}
	ve := &ValidationError{
		ContextName: "test",
		Errors:      errs,
	}

	details := ve.Details()
	require.Len(t, details, 2) // Email (fails email tag) and Age (fails min tag)

	// Find the email detail
	var emailDetail, ageDetail ValidationDetail
	for _, d := range details {
		switch d.Field {
		case "Email":
			emailDetail = d
		case "Age":
			ageDetail = d
		}
	}

	// Email field
	assert.Equal(t, "Email", emailDetail.Field)
	assert.Equal(t, "email", emailDetail.Rule)
	assert.Equal(t, "", emailDetail.Param)
	assert.Equal(t, "not-an-email", emailDetail.Value)
	assert.NotEmpty(t, emailDetail.Message)

	// Age field
	assert.Equal(t, "Age", ageDetail.Field)
	assert.Equal(t, "min", ageDetail.Rule)
	assert.Equal(t, "18", ageDetail.Param)
	assert.Equal(t, 10, ageDetail.Value)
	assert.NotEmpty(t, ageDetail.Message)
}

func TestDetails_RequiredFieldMissing(t *testing.T) {
	v := validator.New()
	err := v.Struct(&testStruct{
		Email: "",
		Age:   25,
		Name:  "",
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	errs := make([]error, len(valErrs))
	for i, fe := range valErrs {
		errs[i] = fe
	}
	ve := &ValidationError{Errors: errs}

	details := ve.Details()
	require.Len(t, details, 2) // Email (required) and Name (required)

	rules := make(map[string]string)
	for _, d := range details {
		rules[d.Field] = d.Rule
	}
	assert.Equal(t, "required", rules["Email"])
	assert.Equal(t, "required", rules["Name"])
}

func TestDetails_FallsBackForNonValidatorErrors(t *testing.T) {
	plainErr := errors.New("something went wrong")
	ve := &ValidationError{
		Errors: []error{plainErr},
	}

	details := ve.Details()
	require.Len(t, details, 1)

	d := details[0]
	assert.Equal(t, "", d.Field)
	assert.Equal(t, "", d.Rule)
	assert.Equal(t, "", d.Param)
	assert.Nil(t, d.Value)
	assert.Equal(t, "something went wrong", d.Message)
}

func TestDetails_MixedErrors(t *testing.T) {
	v := validator.New()
	err := v.Struct(&testStruct{
		Email: "bad",
		Age:   25,
		Name:  "Alice",
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	// Mix validator errors with a plain error
	errs := []error{
		valErrs[0],
		errors.New("custom validation failed"),
	}
	ve := &ValidationError{Errors: errs}

	details := ve.Details()
	require.Len(t, details, 2)

	// First should be the validator field error
	assert.Equal(t, "Email", details[0].Field)
	assert.Equal(t, "email", details[0].Rule)

	// Second should be the plain error fallback
	assert.Equal(t, "", details[1].Field)
	assert.Equal(t, "custom validation failed", details[1].Message)
}

func TestDetails_NilErrors(t *testing.T) {
	ve := &ValidationError{Errors: nil}
	assert.Nil(t, ve.Details())
}

func TestDetails_EmptyErrors(t *testing.T) {
	ve := &ValidationError{Errors: []error{}}
	assert.Nil(t, ve.Details())
}

func TestDetails_PreservesExistingMethods(t *testing.T) {
	// Verify that adding Details() doesn't break Error() or UnderlyingErrors()
	v := validator.New()
	err := v.Struct(&testStruct{
		Email: "bad",
		Age:   5,
		Name:  "",
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	errs := make([]error, len(valErrs))
	for i, fe := range valErrs {
		errs[i] = fe
	}
	ve := &ValidationError{
		ContextName: "cmd",
		Errors:      errs,
	}

	// Error() still works
	errMsg := ve.Error()
	assert.Contains(t, errMsg, "invalid options for cmd")

	// UnderlyingErrors() still works
	underlying := ve.UnderlyingErrors()
	assert.Len(t, underlying, len(valErrs))

	// Details() also works
	details := ve.Details()
	assert.Len(t, details, len(valErrs))
}

func TestDetails_NilErrorInSlice(t *testing.T) {
	ve := &ValidationError{
		Errors: []error{nil, fmt.Errorf("real error")},
	}

	// Should not panic; should skip nil and return 1 detail
	details := ve.Details()
	require.Len(t, details, 1)
	assert.Equal(t, "real error", details[0].Message)
}

func TestDetails_WrappedFieldError(t *testing.T) {
	v := validator.New()
	err := v.Struct(&testStruct{
		Email: "not-an-email",
		Age:   25,
		Name:  "Alice",
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	require.Len(t, valErrs, 1) // only Email fails

	// Wrap the FieldError with fmt.Errorf %w
	wrappedErr := fmt.Errorf("context: %w", valErrs[0])

	ve := &ValidationError{
		Errors: []error{wrappedErr},
	}

	details := ve.Details()
	require.Len(t, details, 1)

	d := details[0]
	assert.Equal(t, "Email", d.Field)
	assert.Equal(t, "email", d.Rule)
	assert.Equal(t, "", d.Param)
	assert.Equal(t, "not-an-email", d.Value)
	assert.NotEmpty(t, d.Message)
}
