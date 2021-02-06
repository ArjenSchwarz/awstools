package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
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

func danglingnodes(cmd *cobra.Command, args []string) {
	resultTitle := "App Mesh Unattached Nodes for mesh " + *meshname
	svc := helpers.AppmeshSession(config.DefaultAwsConfig())
	unserviced := helpers.GetAllUnservicedAppMeshNodes(meshname, svc)
	keys := []string{"Virtual Node"}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	for _, node := range unserviced {
		content := make(map[string]string)
		content["Virtual Node"] = node
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)

}
