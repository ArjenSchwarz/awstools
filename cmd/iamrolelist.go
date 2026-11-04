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
	"log"
	"maps"
	"slices"
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/mermaid"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// rolelistCmd represents the rolelist command
var rolelistCmd = &cobra.Command{
	Use:   "rolelist",
	Short: "Get an overview of the roles and their policies",
	Long: `Retrieves a list of all IAM roles in the account and their policies.
The policies themselves are also shown separately. Attached policies include
their ARN, as policy names are not unique.

The drawio, dot, and mermaid output formats link the roles to policies. In
these formats attached policies are linked by ARN, and dot and mermaid also
use the ARN as the attached policy's node label.`,
	Run: iamrolelist,
}

func iamrolelist(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "IAM Role overview for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	roles, policies := helpers.GetRolesAndPolicies(settings.IsVerbose(), awsConfig.IamClient())
	writeIamrolelist(roles, policies, resultTitle)
}

// writeIamrolelist renders the roles and policies. Policies are keyed by their
// unique ID (ARN for attached policies, "<name> (inline for <role>)" for inline
// policies), as returned by GetRolesAndPolicies. Attached policy names are not
// unique, so graph formats link roles to policies by that ID (T-1656).
func writeIamrolelist(roles []helpers.IAMRole, policies map[string]helpers.IAMPolicyDocument, title string) {
	keys := []string{nameColumn, typeColumn, "ARN", "AssumedFrom", "Policies", "Roles"}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
		keys = append(keys, "DrawioID")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = title
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createIamrolelistDrawIOHeader()
	}
	// DOT and Mermaid use the Name column as node identity.
	nameIsNodeID := output.Settings.NeedsFromToColumns()
	if nameIsNodeID {
		output.Settings.AddFromToColumns(nameColumn, "Policies")
	}
	linkByID := nameIsNodeID || settings.IsDrawIO()
	for _, role := range roles {
		content := make(map[string]any)
		content[nameColumn] = role.Name
		content["AssumedFrom"] = role.CanBeAssumedFrom()
		content[typeColumn] = role.Type
		switch {
		case nameIsNodeID:
			// go-output splits the link column on "," but joins []string
			// with ", " for Mermaid, so pass a pre-joined string.
			content["Policies"] = strings.Join(role.GetPolicyIDs(), ",")
		case linkByID:
			content["Policies"] = role.GetPolicyIDs()
		default:
			content["Policies"] = role.GetPolicyNames()
		}
		if settings.IsDrawIO() {
			content["DrawioID"] = role.ID
			content[imageColumn] = drawio.AWSShape("Security Identity Compliance", "Role")
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for policyID, policy := range policies {
		content := make(map[string]any)
		content[nameColumn] = policyID
		if nameIsNodeID {
			// Without a value go-output renders the missing link column as
			// "%!s(<nil>)" and adds an edge to a non-existent node, which
			// panics in Mermaid.
			content["Policies"] = ""
		}
		if policy.Type == helpers.IAMPolicyTypeAttached {
			content["ARN"] = policyID
			if !nameIsNodeID {
				content[nameColumn] = policy.Name
			}
		}
		if settings.IsDrawIO() {
			content["DrawioID"] = policyID
			content[imageColumn] = drawio.AWSShape("Security Identity Compliance", "Permissions")
		}
		content[typeColumn] = policy.Type
		content["Roles"] = policy.GetRoleNames()
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	writeIamrolelistOutput(output, roles, policies)
}

// writeIamrolelistOutput writes the output, rendering Mermaid destinations
// with iamrolelistMermaid. go-output's own Mermaid renderer re-adds a source
// node for every edge and reuses node IDs, which merges roles that have
// several policies (T-1656). go-output cannot write only the file, so Mermaid
// stdout with a different file format still uses go-output.
func writeIamrolelistOutput(output format.OutputArray, roles []helpers.IAMRole, policies map[string]helpers.IAMPolicyDocument) {
	s := output.Settings
	fileFormat := s.OutputFileFormat
	if fileFormat == "" {
		fileFormat = s.OutputFormat
	}
	stdoutMermaid := s.OutputFormat == "mermaid"
	fileMermaid := s.OutputFile != "" && fileFormat == "mermaid"
	if stdoutMermaid && s.OutputFile != "" && !fileMermaid {
		output.Write()
		return
	}
	if !stdoutMermaid && !fileMermaid {
		output.Write()
		return
	}
	chart := iamrolelistMermaid(roles, policies, s.MermaidSettings)
	if stdoutMermaid {
		printIamrolelistBytes(chart, "", s.S3Bucket)
	} else {
		file := s.OutputFile
		s.OutputFile = ""
		output.Write()
		s.OutputFile = file
	}
	if fileMermaid {
		printIamrolelistBytes(chart, s.OutputFile, s.S3Bucket)
	}
}

func printIamrolelistBytes(contents []byte, file string, bucket format.S3Output) {
	if err := format.PrintByteSlice(contents, file, bucket); err != nil {
		log.Fatal(err.Error())
	}
}

// iamrolelistMermaid renders a flowchart with one node per role and per
// policy ID, and an edge from each role to each of its policies.
func iamrolelistMermaid(roles []helpers.IAMRole, policies map[string]helpers.IAMPolicyDocument, settings *mermaid.Settings) []byte {
	chart := mermaid.NewFlowchart(settings)
	for _, role := range roles {
		chart.AddBasicNode(role.Name)
	}
	policyIDs := slices.Sorted(maps.Keys(policies))
	for _, id := range policyIDs {
		chart.AddBasicNode(id)
	}
	for _, role := range roles {
		ids := role.GetPolicyIDs()
		slices.Sort(ids)
		for _, id := range ids {
			chart.AddEdgeByNames(role.Name, id)
		}
	}
	return []byte(chart.RenderString())
}

// createIamrolelistDrawIOHeader creates and configures the draw.io header settings
func createIamrolelistDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image,DrawioID")
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetIdentity("DrawioID")
	drawioheader.SetLayout(drawio.LayoutHorizontalFlow)
	connection := drawio.NewConnection()
	// Link to DrawioID, not Name: attached policy names are not unique (T-1656).
	connection.From = "Policies"
	connection.To = "DrawioID"
	connection.Invert = false
	connection.Label = "Has Policy"
	drawioheader.AddConnection(connection)
	return drawioheader
}

func init() {
	iamCmd.AddCommand(rolelistCmd)
}
