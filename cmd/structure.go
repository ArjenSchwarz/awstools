package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// structureCmd represents the structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Get a graphical overview of the Organization's structure",
	Long: `This gives you an overview of how the accounts are connected.
	Using the dot output format you can turn this into an image, and
	using drawio you will get a CSV that you can import into draw.io
	with its CSV import functionality.

	Example: awstools organizations structure -o dot | dot -Tpng -o structure.png
	Example: awstools organizations structure -o drawio | pbcopy`,
	Run: orgstructure,
}

func init() {
	organizationsCmd.AddCommand(structureCmd)
}

func orgstructure(cmd *cobra.Command, args []string) {
	switch strings.ToLower(*settings.OutputFormat) {
	case "drawio":
		*settings.Verbose = true
		header := ""
		drawioheader := drawio.DefaultHeader()
		connection := drawio.NewConnection()
		connection.Invert = false
		connection.From = "Children"
		connection.To = "Name"
		drawioheader.AddConnection(connection)
		header = drawioheader.String()
		settings.OutputHeaders = &header
	case "dot":
		dotcolumns := config.DotColumns{
			From: "Name",
			To:   "Children",
		}
		settings.DotColumns = &dotcolumns
	}
	svc := helpers.OrganizationsSession()
	organization := helpers.GetFullOrganization(svc)
	keys := []string{"Name", "Type", "Children"}
	if *settings.Verbose {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys}
	traverseOrgStructureEntry(organization, &output)
	output.Write(*settings)
}

func traverseOrgStructureEntry(entry helpers.OrganizationEntry, output *helpers.OutputArray) {
	content := make(map[string]string)
	content["Name"] = entry.String()
	content["Type"] = entry.Type
	content["Children"] = entry.String()
	if *settings.Verbose {
		content["Image"] = entry.Image
	}
	children := []string{}
	for _, child := range entry.Children {
		children = append(children, child.String())
		traverseOrgStructureEntry(child, output)
	}
	content["Children"] = strings.Join(children, ",")
	holder := helpers.OutputHolder{Contents: content}
	output.AddHolder(holder)
}
