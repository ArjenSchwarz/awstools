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
}
