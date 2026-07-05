package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// structureCmd represents the structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Get a graphical overview of the Organization's structure",
	Long: `This command provides a graphical overview of how the accounts are connected.

Examples:

	awstools organizations structure -o dot | dot -Tpng -o structure.png
	awstools organizations structure -o drawio | pbcopy

Using the dot output format you can turn this into an image, and using drawio you will get a CSV that you can import into draw.io with its CSV import functionality. `,
	RunE: orgstructure,
}

func init() {
	organizationsCmd.AddCommand(structureCmd)
}

func orgstructure(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "AWS Organization Structure"
	organization, err := helpers.GetFullOrganization(awsConfig.OrganizationsClient())
	if err != nil {
		return err
	}
	keys := []string{nameColumn, typeColumn, childrenColumn}
	rows := []output.Record{}
	traverseOrgStructureEntry(organization, &rows)
	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...)).
			Build(),
	}
	if settings.NeedsGraphFormat() {
		docs.Graph = output.New().
			Graph(resultTitle, graphEdges(rows, nameColumn, childrenColumn)).
			Build()
	}
	if settings.IsDrawIO() {
		docs.DrawIO = output.New().
			DrawIO(resultTitle, drawIORecords(rows), createOrganizationsStructureDrawIOHeader()).
			Build()
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

func traverseOrgStructureEntry(entry helpers.OrganizationEntry, rows *[]output.Record) {
	content := make(map[string]any)
	content[nameColumn] = entry.String()
	content[typeColumn] = entry.Type
	if settings.IsDrawIO() {
		imageConversion := map[string]string{
			"ROOT":                awsShape("Management Governance", "Organizations"),
			"ORGANIZATIONAL_UNIT": awsShape("Management Governance", "Organizational Unit"),
			"ACCOUNT":             awsShape("Management Governance", accountColumn),
		}
		content[imageColumn] = imageConversion[entry.Type]
	}
	children := []string{}
	for _, child := range entry.Children {
		children = append(children, child.String())
		traverseOrgStructureEntry(child, rows)
	}
	content[childrenColumn] = children
	*rows = append(*rows, content)
}

func createOrganizationsStructureDrawIOHeader() output.DrawIOHeader {
	drawioheader := drawIOBaseHeader("%Name%", "%Image%", imageColumn)
	drawioheader.Layout = output.DrawIOLayoutVerticalTree
	connection := drawIOConnection()
	connection.Invert = false
	connection.From = childrenColumn
	connection.To = nameColumn
	drawioheader.Connections = append(drawioheader.Connections, connection)
	return drawioheader
}
