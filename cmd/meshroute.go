package cmd

import (
	"strconv"

	"github.com/ArjenSchwarz/awstools/helpers"
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

func meshroute(cmd *cobra.Command, args []string) {
	svc := helpers.AppmeshSession()
	routes := helpers.GetAllAppMeshPaths(meshname, svc)
	keys := []string{"Service", "Path", "Node"}
	if *settings.Verbose {
		keys = append(keys, "Weight")
		keys = append(keys, "Router")
	}
	output := helpers.OutputArray{Keys: keys}
	for _, route := range routes {
		for _, path := range route.VirtualServiceRoutes {
			content := make(map[string]string)
			content["Service"] = route.VirtualServiceName
			content["Path"] = path.Path
			content["Node"] = path.DestinationNode
			if *settings.Verbose {
				content["Weight"] = strconv.Itoa(int(path.Weight))
				content["Router"] = path.Router
			}
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}
