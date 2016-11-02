package cmd

import (
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"
)

// resourcesCmd represents the resources command
var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List all the resources in a stack and any nested stacks",
	Long: `This command will list the resources attached to the provided stack and any possible nested stacks.

	Return values are the ResourceID, Type, and Stack of the resource.

	--verbose will add the status to the output;`,
	Run: listResources,
}

func init() {
	cfnCmd.AddCommand(resourcesCmd)
}

func listResources(cmd *cobra.Command, args []string) {
	resources := helpers.GetNestedCloudFormationResources(stackname)

	keys := []string{"ResourceID", "Type", "Stack"}
	if *settings.Verbose {
		keys = append(keys, "Status")
	}
	output := helpers.OutputArray{Keys: keys}
	for _, resource := range resources {
		content := make(map[string]string)
		content["ResourceID"] = aws.StringValue(resource.PhysicalResourceId)
		content["Type"] = aws.StringValue(resource.ResourceType)
		content["Stack"] = aws.StringValue(resource.StackName)
		if *settings.Verbose {
			content["Status"] = aws.StringValue(resource.ResourceStatus)
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}
