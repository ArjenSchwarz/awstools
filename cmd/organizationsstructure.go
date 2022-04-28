package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/ArjenSchwarz/awstools/lib/format"
	"github.com/ArjenSchwarz/awstools/lib/format/drawio"
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

func orgstructure(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "AWS Organization Structure"
	organization := helpers.GetFullOrganization(awsConfig.OrganizationsClient())
	keys := []string{"Name", "Type", "Children"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createOrganizationsStructureDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("Name", "Children")
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
	content["Children"] = entry.String()
	if settings.IsDrawIO() {
		content["Image"] = imageConversion[entry.Type]
	}
	children := []string{}
	for _, child := range entry.Children {
		children = append(children, child.String())
		traverseOrgStructureEntry(child, output)
	}
	content["Children"] = strings.Join(children, ",")
	holder := format.OutputHolder{Contents: content}
	output.AddHolder(holder)
}

func createOrganizationsStructureDrawIOHeader() drawio.Header {
	drawioheader := drawio.DefaultHeader()
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetLayout(drawio.LayoutVerticalTree)
	connection := drawio.NewConnection()
	connection.Invert = false
	connection.From = "Children"
	connection.To = "Name"
	drawioheader.AddConnection(connection)
	return drawioheader
}
