package structcli_test

import (
	"context"
	"testing"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type validateOnlyOptions struct{}

func (o *validateOnlyOptions) Validate(context.Context) []error { return nil }

type validateWithAttachOptions struct{}

func (o *validateWithAttachOptions) Attach(*cobra.Command) error { return nil }
func (o *validateWithAttachOptions) Validate(context.Context) []error {
	return nil
}

type transformOnlyOptions struct{}

func (o *transformOnlyOptions) Transform(context.Context) error { return nil }

type transformWithAttachOptions struct{}

func (o *transformWithAttachOptions) Attach(*cobra.Command) error { return nil }
func (o *transformWithAttachOptions) Transform(context.Context) error {
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
