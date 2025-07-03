package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
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
	Run: orgstructure,
}

func init() {
	organizationsCmd.AddCommand(structureCmd)
}

func orgstructure(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "AWS Organization Structure"
	organization := helpers.GetFullOrganization(awsConfig.OrganizationsClient())
	keys := []string{"Name", "Type", childrenColumn}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createOrganizationsStructureDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("Name", childrenColumn)
	}
	traverseOrgStructureEntry(organization, &output)
	output.Write()
}

func traverseOrgStructureEntry(entry helpers.OrganizationEntry, output *format.OutputArray) {
	imageConversion := map[string]string{
		"ROOT":                drawio.AWSShape("Management Governance", "Organizations"),
		"ORGANIZATIONAL_UNIT": drawio.AWSShape("Management Governance", "Organizational Unit"),
		"ACCOUNT":             drawio.AWSShape("Management Governance", "Account"),
	}
	content := make(map[string]interface{})
	content["Name"] = entry.String()
	content["Type"] = entry.Type
	content[childrenColumn] = entry.String()
	if settings.IsDrawIO() {
		content["Image"] = imageConversion[entry.Type]
	}
	children := []string{}
	for _, child := range entry.Children {
		children = append(children, child.String())
		traverseOrgStructureEntry(child, output)
	}
	content[childrenColumn] = children
	holder := format.OutputHolder{Contents: content}
	output.AddHolder(holder)
}

func createOrganizationsStructureDrawIOHeader() drawio.Header {
	drawioheader := drawio.DefaultHeader()
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetLayout(drawio.LayoutVerticalTree)
	connection := drawio.NewConnection()
	connection.Invert = false
	connection.From = childrenColumn
	connection.To = nameColumn
	drawioheader.AddConnection(connection)
	return drawioheader
}
