package structcli_test

import (
	"context"
	"testing"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// --- Standalone interface test types ---

type validateOnlyOptions struct{}

func (o *validateOnlyOptions) Validate(context.Context) []error { return nil }

type transformOnlyOptions struct{}

func (o *transformOnlyOptions) Transform(context.Context) error { return nil }

type contextInjectorOnly struct{}

func (o *contextInjectorOnly) Context(ctx context.Context) context.Context { return ctx }

// --- Deprecated interface test types (with Attach) ---

type validateWithAttachOptions struct{}

func (o *validateWithAttachOptions) Attach(*cobra.Command) error { return nil }
func (o *validateWithAttachOptions) Validate(context.Context) []error {
	return nil
}

type transformWithAttachOptions struct{}

func (o *transformWithAttachOptions) Attach(*cobra.Command) error { return nil }
func (o *transformWithAttachOptions) Transform(context.Context) error {
	return nil
}

type contextOptionsWithAttach struct{}

func (o *contextOptionsWithAttach) Attach(*cobra.Command) error              { return nil }
func (o *contextOptionsWithAttach) Context(ctx context.Context) context.Context { return ctx }
func (o *contextOptionsWithAttach) FromContext(context.Context) error         { return nil }

// --- Types implementing both standalone and deprecated ---

type validateBothInterfaces struct{}

func (o *validateBothInterfaces) Attach(*cobra.Command) error { return nil }
func (o *validateBothInterfaces) Validate(context.Context) []error {
	return nil
}

func TestOptionContracts(t *testing.T) {
	t.Run("ValidatableOptions requires Options", func(t *testing.T) {
		var validateOnly any = &validateOnlyOptions{}
		var validateWithAttach any = &validateWithAttachOptions{}

		_, validateOnlyImplements := validateOnly.(structcli.ValidatableOptions)
		_, validateWithAttachImplements := validateWithAttach.(structcli.ValidatableOptions)

		assert.False(t, validateOnlyImplements)
		assert.True(t, validateWithAttachImplements)
	})

	t.Run("TransformableOptions requires Options", func(t *testing.T) {
		var transformOnly any = &transformOnlyOptions{}
		var transformWithAttach any = &transformWithAttachOptions{}

		_, transformOnlyImplements := transformOnly.(structcli.TransformableOptions)
		_, transformWithAttachImplements := transformWithAttach.(structcli.TransformableOptions)

		assert.False(t, transformOnlyImplements)
		assert.True(t, transformWithAttachImplements)
	})
}

func TestStandaloneCapabilityInterfaces(t *testing.T) {
	t.Run("Validatable does not require Attach", func(t *testing.T) {
		var v any = &validateOnlyOptions{}
		_, ok := v.(structcli.Validatable)
		assert.True(t, ok, "validateOnlyOptions should implement Validatable")

		_, hasAttach := v.(structcli.Options)
		assert.False(t, hasAttach, "validateOnlyOptions should not implement Options")
	})

	t.Run("Transformable does not require Attach", func(t *testing.T) {
		var v any = &transformOnlyOptions{}
		_, ok := v.(structcli.Transformable)
		assert.True(t, ok, "transformOnlyOptions should implement Transformable")

		_, hasAttach := v.(structcli.Options)
		assert.False(t, hasAttach, "transformOnlyOptions should not implement Options")
	})

	t.Run("ContextInjector does not require Attach", func(t *testing.T) {
		var v any = &contextInjectorOnly{}
		_, ok := v.(structcli.ContextInjector)
		assert.True(t, ok, "contextInjectorOnly should implement ContextInjector")

		_, hasAttach := v.(structcli.Options)
		assert.False(t, hasAttach, "contextInjectorOnly should not implement Options")

		_, hasFromContext := v.(structcli.ContextOptions)
		assert.False(t, hasFromContext, "contextInjectorOnly should not implement ContextOptions")
	})

	t.Run("ContextInjector does not require FromContext", func(t *testing.T) {
		var v any = &contextInjectorOnly{}
		_, ok := v.(structcli.ContextInjector)
		assert.True(t, ok)
		// ContextInjector only has Context(), not FromContext()
	})

	t.Run("deprecated interfaces still satisfied by types with Attach", func(t *testing.T) {
		var v any = &validateBothInterfaces{}

		_, isValidatable := v.(structcli.Validatable)
		_, isValidatableOptions := v.(structcli.ValidatableOptions)
		assert.True(t, isValidatable, "should implement Validatable")
		assert.True(t, isValidatableOptions, "should still implement ValidatableOptions")
	})

	t.Run("ContextOptions type also satisfies ContextInjector", func(t *testing.T) {
		var v any = &contextOptionsWithAttach{}

		_, isInjector := v.(structcli.ContextInjector)
		_, isContextOptions := v.(structcli.ContextOptions)
		assert.True(t, isInjector, "ContextOptions implementor should also satisfy ContextInjector")
		assert.True(t, isContextOptions, "should still implement ContextOptions")
	})
}
