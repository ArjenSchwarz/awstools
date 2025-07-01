package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// meshrouteCmd represents the meshroute command
var meshrouteCmd = &cobra.Command{
	Use:   "routelist",
	Short: "Get an overview of all routes in the mesh",
	Long:  `This displays all the routes in the mesh`,
	Run:   meshroute,
}

func init() {
	appmeshCmd.AddCommand(meshrouteCmd)
}

func meshroute(_ *cobra.Command, _ []string) {
	resultTitle := "Overview of the routes in the mesh"
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.AppmeshClient()
	routes := helpers.GetAllAppMeshPaths(meshname, svc)
	keys := []string{"Service", "Path", "Node"}
	if settings.IsVerbose() {
		keys = append(keys, "Weight")
		keys = append(keys, "Router")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, route := range routes {
		for _, path := range route.VirtualServiceRoutes {
			content := make(map[string]interface{})
			content["Service"] = route.VirtualServiceName
			content["Path"] = path.Path
			content["Node"] = path.DestinationNode
			if settings.IsVerbose() {
				content["Weight"] = int(path.Weight)
				content["Router"] = path.Router
			}
			holder := format.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write()
}
