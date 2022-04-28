// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// rolelistCmd represents the rolelist command
var rolelistCmd = &cobra.Command{
	Use:   "rolelist",
	Short: "Get an overview of the roles and their policies",
	Long: `Retrieves a list of all IAM roles in the account and their policies.
The policies themselves are also shown separately.

The drawio output format links the users to policies.`,
	Run: iamrolelist,
}

func iamrolelist(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "IAM Role overview for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	roles, policies := helpers.GetRolesAndPolicies(settings.IsVerbose(), awsConfig.IamClient())
	keys := []string{"Name", "Type", "AssumedFrom", "Policies", "Roles"}
	stringSeparator := ", "
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
		keys = append(keys, "DrawioID")
		stringSeparator = ","
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createIamrolelistDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("Name", "Policies")
	}
	for _, role := range roles {
		content := make(map[string]interface{})
		content["Name"] = role.Name
		content["AssumedFrom"] = strings.Join(role.CanBeAssumedFrom(), stringSeparator)
		content["Type"] = role.Type
		content["Policies"] = strings.Join(role.GetPolicyNames(), stringSeparator)
		if settings.IsDrawIO() {
			content["DrawioID"] = role.ID
			content["Image"] = drawio.AWSShape("Security Identity Compliance", "Role")
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for policyname, policy := range policies {
		content := make(map[string]interface{})
		content["Name"] = policyname
		if settings.IsDrawIO() {
			content["DrawioID"] = policyname
			content["Image"] = drawio.AWSShape("Security Identity Compliance", "Permissions")
		}
		content["Type"] = policy.Type
		content["Roles"] = strings.Join(policy.GetRoleNames(), stringSeparator)
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

// createIamrolelistDrawIOHeader creates and configures the draw.io header settings
func createIamrolelistDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image,DrawioID")
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetIdentity("DrawioID")
	drawioheader.SetLayout(drawio.LayoutHorizontalFlow)
	connection := drawio.NewConnection()
	connection.From = "Policies"
	connection.To = "Name"
	connection.Invert = false
	connection.Label = "Has Policy"
	drawioheader.AddConnection(connection)
	return drawioheader
}

func init() {
	iamCmd.AddCommand(rolelistCmd)
}
