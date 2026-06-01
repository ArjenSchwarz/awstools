package cmd

import (
	"github.com/spf13/cobra"
)

// appmeshCmd represents the appmesh command
var appmeshCmd = &cobra.Command{
	Use:   "appmesh",
	Short: "App Mesh commands",
	Long:  `This lets you run various commands for AWS App Mesh`,
}

var meshname *string

func init() {
	rootCmd.AddCommand(appmeshCmd)
	meshname = appmeshCmd.PersistentFlags().StringP("meshname", "m", "", "The name of the mesh")
	// Every appmesh subcommand operates on a specific mesh, so require the flag.
	// Cobra rejects the command with a usage error before the Run function (and
	// thus any AWS config loading or API call) executes. See T-1382.
	if err := appmeshCmd.MarkPersistentFlagRequired("meshname"); err != nil {
		panic(err)
	}
}
