package cmd

import (
	"encoding/json"
	"log"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// structureCmd represents the structure command
var orgnamesCmd = &cobra.Command{
	Use:   "names",
	Short: "Get a list of the account names useful for name files",
	Long: `This command provides a list of the account names for use in a name file in cases where no aliases are set.

Examples:

	awstools organizations names -o json`,
	Run: orgnames,
}

func init() {
	organizationsCmd.AddCommand(orgnamesCmd)
}

func orgnames(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	organization := helpers.GetFullOrganization(awsConfig.OrganizationsClient())
	result := make(map[string]string)
	result = traverseOrgStructureEntryForNames(organization, result)
	jsonString, _ := json.Marshal(result)
	err := format.PrintByteSlice(jsonString, settings.GetString("output.file"), format.NewOutputSettings().S3Bucket)
	if err != nil {
		log.Fatal(err.Error())
	}
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
