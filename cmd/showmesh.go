package cmd

import (
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// showmeshCmd represents the showmesh command
var showmeshCmd = &cobra.Command{
	Use:   "showmesh",
	Short: "Show the connections between virtual nodes",
	Long: `You can see which nodes are allowed access to which other nodes
	based on the current App Mesh configuration. If you use the dot output
	format you can use various tools to output this into an image.

	Example:
	awstools appmesh showmesh -m bookinfo-mesh -o dot | dot -Tpng  > bookinfo-mesh.png`,
	Run: showmesh,
}

func init() {
	appmeshCmd.AddCommand(showmeshCmd)
}

func showmesh(cmd *cobra.Command, args []string) {
	// Step 1: Get all paths for each service
	svc := helpers.AppmeshSession()
	nodes := helpers.GetAllAppMeshNodeConnections(meshname, svc)
	keys := []string{"From", "To"}
	output := helpers.OutputArray{Keys: keys}
	for _, node := range nodes {
		if len(node.BackendNodes) == 0 {
			content := make(map[string]string)
			content["From"] = node.VirtualNodeName
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
		for _, backendNode := range node.BackendNodes {
			content := make(map[string]string)
			content["From"] = node.VirtualNodeName
			content["To"] = backendNode
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}
