package cmd

import (
	"encoding/json"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// structureCmd represents the structure command
var orgnamesCmd = &cobra.Command{
	Use:   "names",
	Short: "Get a list of the account names useful for name files",
	Long: `This command provides a list of the account names for use in a name file in cases where no aliases are set.

Examples:

	awstools organizations names -o json`,
	RunE: orgnames,
}

func init() {
	organizationsCmd.AddCommand(orgnamesCmd)
}

func orgnames(_ *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	organization, err := helpers.GetFullOrganization(awsConfig.OrganizationsClient())
	if err != nil {
		return err
	}
	result := make(map[string]string)
	result = traverseOrgStructureEntryForNames(organization, result)
	jsonString, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return writeToFileOrStdout(jsonString, settings.GetString("output.file"))
}

func traverseOrgStructureEntryForNames(entry helpers.OrganizationEntry, output map[string]string) map[string]string {
	if entry.Type == "ACCOUNT" {
		output[entry.ID] = entry.Name
	}
	for _, child := range entry.Children {
		traverseOrgStructureEntryForNames(child, output)
	}
	return output
}
