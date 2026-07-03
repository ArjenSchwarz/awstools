package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
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
	RunE: showmesh,
}

func init() {
	appmeshCmd.AddCommand(showmeshCmd)
}

func showmesh(cmd *cobra.Command, _ []string) error {
	resultTitle := "Virtual node connections for mesh " + *meshname
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.AppmeshClient()
	nodes := helpers.GetAllAppMeshNodeConnections(meshname, svc)
	keys := []string{nameColumn, "Endpoints"}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
	}

	rows := []output.Record{}
	for _, node := range nodes {
		content := make(map[string]any)
		content[nameColumn] = node.VirtualNodeName
		if settings.IsDrawIO() {
			content[imageColumn] = awsShape("Containers", "Container")
		}
		endpoints := append([]string{}, node.BackendNodes...)
		content["Endpoints"] = endpoints
		rows = append(rows, content)
	}

	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...)).
			Build(),
	}
	if settings.NeedsGraphFormat() {
		graphRows := make([]map[string]any, len(rows))
		for i, row := range rows {
			graphRows[i] = row
		}
		docs.Graph = output.New().
			Graph(resultTitle, graphEdges(graphRows, nameColumn, "Endpoints")).
			Build()
	}
	if settings.IsDrawIO() {
		docs.DrawIO = output.New().
			DrawIO(resultTitle, rows, createAppmeshShowmeshDrawIOHeader()).
			Build()
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

func createAppmeshShowmeshDrawIOHeader() output.DrawIOHeader {
	header := drawIOBaseHeader("%Name%", "%Image%", imageColumn)
	connection := drawIOConnection()
	connection.From = "Endpoints"
	connection.To = nameColumn
	connection.Invert = false
	connection.Label = "Calls"
	header.Connections = append(header.Connections, connection)
	return header
}
