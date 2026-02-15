package internalusage

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroups(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().String("plain", "", "plain flag")
	cmd.Flags().String("grouped", "", "grouped flag")
	cmd.Flags().String("annotated-no-group", "", "annotated without group")
	cmd.PersistentFlags().Bool("config", false, "config flag")

	err := cmd.Flags().SetAnnotation("grouped", FlagGroupAnnotation, []string{"Alpha"})
	require.NoError(t, err)
	err = cmd.Flags().SetAnnotation("annotated-no-group", "other", []string{"x"})
	require.NoError(t, err)

	groups := Groups(cmd)

	require.Contains(t, groups, localGroupID)
	require.Contains(t, groups, "Alpha")
	require.Contains(t, groups, globalGroupID)

	assert.NotNil(t, groups[localGroupID].Lookup("plain"))
	assert.NotNil(t, groups[localGroupID].Lookup("annotated-no-group"))
	assert.NotNil(t, groups["Alpha"].Lookup("grouped"))
	assert.NotNil(t, groups[globalGroupID].Lookup("config"))
}
