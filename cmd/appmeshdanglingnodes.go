package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// danglingnodesCmd represents the danglingnodes command
var danglingnodesCmd = &cobra.Command{
	Use:   "danglingnodes",
	Short: "Get all dangling nodes",
	Long:  `Get an overview of all nodes without a route or service attached to them`,
	RunE:  danglingnodes,
}

func init() {
	appmeshCmd.AddCommand(danglingnodesCmd)
}

func danglingnodes(cmd *cobra.Command, _ []string) error {
	resultTitle := "App Mesh Unattached Nodes for mesh " + *meshname
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.AppmeshClient()
	unserviced := helpers.GetAllUnservicedAppMeshNodes(meshname, svc)
	keys := []string{"Virtual Node"}
	rows := []map[string]any{}
	for _, node := range unserviced {
		content := make(map[string]any)
		content["Virtual Node"] = node
		rows = append(rows, content)
	}
	doc := output.New().
		Table(resultTitle, rows, output.WithKeys(keys...)).
		Build()
	return settings.RenderDocument(cmd.Context(), doc)
}
