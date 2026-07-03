package cmd

import (
	"encoding/json"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// namesCmd represents the names command
var namesCmd = &cobra.Command{
	Use:   "names",
	Short: "Get the names for the resources in the account",
	Long: `These names can be stored in a file and then used by other functionalities.
	This is especially useful for commands that deal with multiple accounts.

	Only outputs as JSON.`,
	RunE: names,
}

func init() {
	rootCmd.AddCommand(namesCmd)
}

func names(_ *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	var names []map[string]string
	if settings.ShouldCombineAndAppend() {
		names = append(names, helpers.GetStringMapFromJSONFile(settings.GetString("output.file")))
	}
	names = append(names, helpers.GetAllEC2ResourceNames(awsConfig.Ec2Client(), awsConfig.DirectConnectClient()))
	names = append(names, helpers.GetAllRdsResourceNames(awsConfig.RdsClient()))
	names = append(names, helpers.GetAccountAlias(awsConfig.IamClient(), awsConfig.StsClient()))
	allNames := helpers.FlattenStringMaps(names)
	jsonString, err := json.Marshal(allNames)
	if err != nil {
		return err
	}
	return writeToFileOrStdout(jsonString, settings.GetString("output.file"))
}

// writeToFileOrStdout writes the contents to the provided file when one is
// set, otherwise it prints the contents to stdout. This mirrors v1's
// PrintByteSlice semantics: a single destination, never both.
func writeToFileOrStdout(contents []byte, outputFile string) error {
	if outputFile == "" {
		_, err := os.Stdout.Write(contents)
		return err
	}
	return os.WriteFile(outputFile, contents, 0o666)
}
