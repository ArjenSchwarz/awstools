package cmd

import (
	"encoding/json"
	"log"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

var appendTo *bool

// namesCmd represents the names command
var namesCmd = &cobra.Command{
	Use:   "names",
	Short: "Get the names for the resources in the account",
	Long: `These names can be stored in a file and then used by other functionalities.
	This is especially useful for commands that deal with multiple accounts.

	Only outputs as JSON.`,
	Run: names,
}

func init() {
	rootCmd.AddCommand(namesCmd)
}

func names(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig()
	var names []map[string]string
	if settings.ShouldCombineAndAppend() {
		names = append(names, helpers.GetStringMapFromJSONFile(*settings.OutputFile))
	}
	names = append(names, helpers.GetAllEC2ResourceNames(awsConfig.Ec2Client()))
	names = append(names, helpers.GetAllRdsResourceNames(awsConfig.RdsClient()))
	names = append(names, helpers.GetAccountAlias(awsConfig.IamClient(), awsConfig.StsClient()))
	allNames := helpers.FlattenStringMaps(names)
	jsonString, _ := json.Marshal(allNames)
	err := helpers.PrintByteSlice(jsonString, *settings.OutputFile)
	if err != nil {
		log.Fatal(err.Error())
	}
}
