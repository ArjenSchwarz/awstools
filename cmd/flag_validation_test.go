package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequiredFlagsRejectedBeforeAWSCalls verifies that the cfn and appmesh
// subcommands reject a missing/empty required flag (--stack / --meshname) with
// a Cobra usage error, before any AWS config is loaded or AWS API is called.
//
// Bug T-1382: --stack and --meshname were registered with an empty-string
// default and not marked required, so the subcommands ran their Run function
// with an empty pointer, eventually sending an empty StackName/meshName to AWS
// and panicking/printing an AWS validation error instead of Cobra usage.
//
// Expected: each command returns an error containing "required flag(s)" and the
// flag name, with no AWS interaction. If this test reaches an AWS call it would
// fail (panic or hang) rather than returning a clean required-flag error.
func TestRequiredFlagsRejectedBeforeAWSCalls(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flagName string
	}{
		{
			name:     "cfn resources without --stack",
			args:     []string{"cfn", "resources"},
			flagName: "stack",
		},
		{
			name:     "appmesh routelist without --meshname",
			args:     []string{"appmesh", "routelist"},
			flagName: "meshname",
		},
		{
			name:     "appmesh showmesh without --meshname",
			args:     []string{"appmesh", "showmesh"},
			flagName: "meshname",
		},
		{
			name:     "appmesh danglingnodes without --meshname",
			args:     []string{"appmesh", "danglingnodes"},
			flagName: "meshname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			require.Error(t, err, "expected a required-flag error, command should not reach AWS")
			assert.Contains(t, err.Error(), "required flag(s)")
			assert.Contains(t, err.Error(), tt.flagName)
		})
	}
}
