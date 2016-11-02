// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
