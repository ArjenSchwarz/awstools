package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

// resourcesCmd represents the resources command
var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List all the resources in a stack and any nested stacks",
	Long: `This command will list the resources attached to the provided stack and any possible nested stacks.

Return values are the ResourceID, Type, and Stack of the resource. You can use the --namefile flag to show names instead of resource ids.

--verbose will add the status and logicalname (the nme within the stack) to the output`,
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

// buildCfnResource converts a CloudFormation StackResource into the local
// cfnResource view used for output. It is nil-safe for every optional SDK
// pointer field and falls back to the logical resource id (or an empty
// string) when PhysicalResourceId is absent — AWS returns a nil
// PhysicalResourceId for resources that have not been created yet and for
// resource types that never emit one. See T-733.
func buildCfnResource(resource types.StackResource, nameResolver func(string) string) cfnResource {
	physicalID := aws.ToString(resource.PhysicalResourceId)
	logicalID := aws.ToString(resource.LogicalResourceId)

	resourceName := logicalID
	if physicalID != "" {
		resourceName = nameResolver(physicalID)
	}

	return cfnResource{
		ResourceID:   physicalID,
		Type:         aws.ToString(resource.ResourceType),
		Stack:        aws.ToString(resource.StackName),
		Status:       string(resource.ResourceStatus),
		LogicalName:  logicalID,
		ResourceName: resourceName,
	}
}

func listResources(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "CloudFormation resources for stack " + *stackname
	unparsedResources := helpers.GetNestedCloudFormationResources(stackname, awsConfig.CloudformationClient())
	resources := make([]cfnResource, len(unparsedResources))

	c := make(chan cfnResource)
	for _, unparsedResource := range unparsedResources {
		go func(resource types.StackResource) {
			c <- buildCfnResource(resource, getName)
		}(unparsedResource)
	}
	for i := range unparsedResources {
		resources[i] = <-c
	}
	keys := []string{"ResourceID", "Type", "Stack", "Name"}
	if settings.IsVerbose() {
		keys = append(keys, "Status")
		keys = append(keys, "LogicalName")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, resource := range resources {
		content := make(map[string]any)
		content["ResourceID"] = resource.ResourceID
		content["Type"] = resource.Type
		content["Stack"] = resource.Stack
		content["Name"] = resource.ResourceName
		if settings.IsVerbose() {
			content["Status"] = resource.Status
			content["LogicalName"] = resource.LogicalName
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}
