package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// danglingnodesCmd represents the danglingnodes command
var danglingnodesCmd = &cobra.Command{
	Use:   "danglingnodes",
	Short: "Get all dangling nodes",
	Long:  `Get an overview of all nodes without a route or service attached to them`,
	Run:   danglingnodes,
}

func init() {
	appmeshCmd.AddCommand(danglingnodesCmd)
}

func danglingnodes(_ *cobra.Command, _ []string) {
	resultTitle := "App Mesh Unattached Nodes for mesh " + *meshname
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.AppmeshClient()
	unserviced := helpers.GetAllUnservicedAppMeshNodes(meshname, svc)
	keys := []string{"Virtual Node"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, node := range unserviced {
		content := make(map[string]any)
		content["Virtual Node"] = node
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()

}
