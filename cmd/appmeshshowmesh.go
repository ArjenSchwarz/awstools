package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/ArjenSchwarz/awstools/lib/format"
	"github.com/ArjenSchwarz/awstools/lib/format/drawio"
	"github.com/spf13/cobra"
)

// showmeshCmd represents the showmesh command
var showmeshCmd = &cobra.Command{
	Use:   "showmesh",
	Short: "Show the connections between virtual nodes",
	Long: `You can see which nodes are allowed access to which other nodes based on the current App Mesh configuration.

Example:

	awstools appmesh showmesh -m bookinfo-mesh -o dot | dot -Tpng  -o bookinfo-mesh.png
	awstools appmesh showmesh -m bookinfo-mesh -o drawio | pbcopy

Using the dot output format you can turn this into an image, and using drawio you will get a CSV that you can import into draw.io with its CSV import functionality
`,
	Run: showmesh,
}

func init() {
	appmeshCmd.AddCommand(showmeshCmd)
}

func showmesh(cmd *cobra.Command, args []string) {
	resultTitle := "Virtual node connections for mesh " + *meshname
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.AppmeshClient()
	nodes := helpers.GetAllAppMeshNodeConnections(meshname, svc)
	keys := []string{"Name", "Endpoints"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	// Set output specific config
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createAppmeshShowmeshDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("Name", "Endpoints")
	}

	for _, node := range nodes {
		content := make(map[string]interface{})
		content["Name"] = node.VirtualNodeName
		if settings.IsDrawIO() {
			content["Image"] = drawio.AWSShape("Containers", "Container")
		}
		endpoints := []string{}
		for _, backendNode := range node.BackendNodes {
			endpoints = append(endpoints, backendNode)
		}
		content["Endpoints"] = strings.Join(endpoints, ",")
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

func createAppmeshShowmeshDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = "Endpoints"
	connection.To = "Name"
	connection.Invert = false
	connection.Label = "Calls"
	drawioheader.AddConnection(connection)
	return drawioheader
}
