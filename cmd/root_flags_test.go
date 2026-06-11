package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileFormatFlagRegistration verifies that --file-format is registered as
// a global persistent flag on the root command and bound to the
// output.file-format viper key, so every subcommand can write the --file
// output in a different format than stdout (T-1535).
func TestFileFormatFlagRegistration(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("file-format")
	require.NotNil(t, flag, "--file-format must be a persistent flag on the root command")
	assert.Equal(t, "", flag.DefValue, "--file-format must default to empty so the file falls back to the --output format")

	require.NoError(t, flag.Value.Set("csv"))
	t.Cleanup(func() {
		// Restore the flag instead of viper.Reset(), which would tear down
		// the package-wide flag bindings created in init().
		_ = flag.Value.Set("")
		flag.Changed = false
	})
	assert.Equal(t, "csv", viper.GetString("output.file-format"), "--file-format must be bound to the output.file-format viper key")
}
