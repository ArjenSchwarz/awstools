package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

// resourcesCmd represents the resources command
var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List all the resources in a stack and any nested stacks",
	Long: `This command will list the resources attached to the provided stack and any possible nested stacks.

Return values are the ResourceID, Type, and Stack of the resource.

--verbose will add the status and logicalname (the nme within the stack) to the output;`,
	Run: listResources,
}

type cfnResource struct {
	ResourceID   string
	Type         string
	Stack        string
	Status       string
	ResourceName string
	LogicalName  string
}

func init() {
	cfnCmd.AddCommand(resourcesCmd)
}

func listResources(cmd *cobra.Command, args []string) {
	unparsedResources := helpers.GetNestedCloudFormationResources(stackname, config.DefaultAwsConfig())
	resources := make([]cfnResource, len(unparsedResources))

	c := make(chan cfnResource)
	for _, unparsedResource := range unparsedResources {
		go func(resource cloudformation.StackResource) {
			resourceStruct := cfnResource{
				ResourceID:   aws.StringValue(resource.PhysicalResourceId),
				Type:         aws.StringValue(resource.ResourceType),
				Stack:        aws.StringValue(resource.StackName),
				Status:       string(resource.ResourceStatus),
				LogicalName:  aws.StringValue(resource.LogicalResourceId),
				ResourceName: aws.StringValue(resource.PhysicalResourceId),
			}
			// Override the resource name when there is a better name available
			switch resourceStruct.Type {
			case "AWS::EC2::Instance":
				resourceStruct.ResourceName = helpers.GetEc2Name(resource.PhysicalResourceId)
			case "AWS::RDS::DBInstance":
				resourceStruct.ResourceName = helpers.GetRDSName(resource.PhysicalResourceId)

			}
			c <- resourceStruct
		}(unparsedResource)
	}
	for i := 0; i < len(unparsedResources); i++ {
		resources[i] = <-c
	}
	keys := []string{"ResourceID", "Type", "Stack", "Name"}
	if *settings.Verbose {
		keys = append(keys, "Status")
		keys = append(keys, "LogicalName")
	}
	output := helpers.OutputArray{Keys: keys}
	for _, resource := range resources {
		content := make(map[string]string)
		content["ResourceID"] = resource.ResourceID
		content["Type"] = resource.Type
		content["Stack"] = resource.Stack
		content["Name"] = resource.ResourceName
		if *settings.Verbose {
			content["Status"] = resource.Status
			content["LogicalName"] = resource.LogicalName
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}
