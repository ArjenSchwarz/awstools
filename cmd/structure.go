package cmd

import (
	"strings"

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

func orgDotstructure(cmd *cobra.Command, args []string) {
	svc := helpers.OrganizationsSession()
	organization := helpers.GetFullOrganization(svc)
	keys := []string{"From", "To"}
	output := helpers.OutputArray{Keys: keys}
	traverseOrgDotStructureEntry(organization, &output)
	output.Write(*settings)
}

func traverseOrgDotStructureEntry(entry helpers.OrganizationEntry, output *helpers.OutputArray) {
	for _, child := range entry.Children {
		content := make(map[string]string)
		content["From"] = entry.String()
		content["To"] = child.String()
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
		traverseOrgDotStructureEntry(child, output)
	}
	if len(entry.Children) == 0 {
		content := make(map[string]string)
		content["From"] = entry.String()
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
}

func orgstructure(cmd *cobra.Command, args []string) {
	if strings.ToLower(*settings.OutputFormat) == "dot" {
		orgDotstructure(cmd, args)
		return
	}
	if strings.ToLower(*settings.OutputFormat) == "drawio" {
		*settings.Verbose = true
	}
	svc := helpers.OrganizationsSession()
	organization := helpers.GetFullOrganization(svc)
	keys := []string{"Name", "Type", "Parent"}
	if *settings.Verbose {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys}
	content := make(map[string]string)
	content["Name"] = organization.String()
	content["Type"] = organization.Type
	if *settings.Verbose {
		content["Image"] = organization.Image
	}
	holder := helpers.OutputHolder{Contents: content}
	output.AddHolder(holder)
	traverseOrgStructureEntry(organization, &output)
	output.Write(*settings)
}

func traverseOrgStructureEntry(entry helpers.OrganizationEntry, output *helpers.OutputArray) {
	for _, child := range entry.Children {
		content := make(map[string]string)
		content["Name"] = child.String()
		content["Type"] = child.Type
		content["Parent"] = entry.String()
		if *settings.Verbose {
			content["Image"] = child.Image
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
		traverseOrgStructureEntry(child, output)
	}
}
