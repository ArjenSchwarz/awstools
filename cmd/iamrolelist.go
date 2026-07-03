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
	output "github.com/ArjenSchwarz/go-output/v2"

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
	RunE: iamrolelist,
}

func iamrolelist(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "IAM Role overview for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	roles, policies := helpers.GetRolesAndPolicies(settings.IsVerbose(), awsConfig.IamClient())
	keys := []string{nameColumn, typeColumn, "AssumedFrom", "Policies", "Roles"}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
		keys = append(keys, "DrawioID")
	}
	rows := []output.Record{}
	for _, role := range roles {
		content := make(map[string]any)
		content[nameColumn] = role.Name
		content["AssumedFrom"] = role.CanBeAssumedFrom()
		content[typeColumn] = role.Type
		content["Policies"] = role.GetPolicyNames()
		if settings.IsDrawIO() {
			content["DrawioID"] = role.ID
			content[imageColumn] = awsShape("Security Identity Compliance", "Role")
		}
		rows = append(rows, content)
	}
	for policyname, policy := range policies {
		content := make(map[string]any)
		content[nameColumn] = policyname
		if settings.IsDrawIO() {
			content["DrawioID"] = policyname
			content[imageColumn] = awsShape("Security Identity Compliance", "Permissions")
		}
		content[typeColumn] = policy.Type
		content["Roles"] = policy.GetRoleNames()
		rows = append(rows, content)
	}

	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...)).
			Build(),
	}
	if settings.NeedsGraphFormat() {
		graphRows := make([]map[string]any, len(rows))
		for i, row := range rows {
			graphRows[i] = row
		}
		docs.Graph = output.New().
			Graph(resultTitle, graphEdges(graphRows, nameColumn, "Policies")).
			Build()
	}
	if settings.IsDrawIO() {
		docs.DrawIO = output.New().
			DrawIO(resultTitle, rows, createIamrolelistDrawIOHeader()).
			Build()
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

// createIamrolelistDrawIOHeader creates and configures the draw.io header settings
func createIamrolelistDrawIOHeader() output.DrawIOHeader {
	header := drawIOBaseHeader("%Name%", "%Image%", "Image,DrawioID")
	header.Identity = "DrawioID"
	header.Layout = output.DrawIOLayoutHorizontalFlow
	connection := drawIOConnection()
	connection.From = "Policies"
	connection.To = nameColumn
	connection.Invert = false
	connection.Label = "Has Policy"
	header.Connections = append(header.Connections, connection)
	return header
}

func init() {
	iamCmd.AddCommand(rolelistCmd)
}
