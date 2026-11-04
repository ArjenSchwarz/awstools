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
	"bytes"
	"html/template"
	"log"
	"maps"
	"os"
	"slices"
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/ArjenSchwarz/go-output/templates"

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
	outputSettings := settings.NewOutputSettings()
	outputSettings.Title = resultTitle
	writeIamrolelist(roles, policies, outputSettings)
}

// writeIamrolelist renders the roles and policies. Policies are keyed by their
// unique ID (ARN for attached policies, "<name> (inline for <role>)" for inline
// policies), as returned by GetRolesAndPolicies. Attached policy names are not
// unique, so graph formats link roles to policies by that ID (T-1656).
//
// go-output's Mermaid renderer re-adds a source node for every edge and reuses
// node IDs, which merges roles that have several policies. Any destination
// (stdout or file) that asks for Mermaid is therefore rendered with
// iamrolelistMermaid, and the other destination keeps its own format.
func writeIamrolelist(roles []helpers.IAMRole, policies map[string]helpers.IAMPolicyDocument, s *format.OutputSettings) {
	fileFormat := s.OutputFileFormat
	if fileFormat == "" {
		fileFormat = s.OutputFormat
	}
	stdoutMermaid := s.OutputFormat == "mermaid"
	fileMermaid := s.OutputFile != "" && fileFormat == "mermaid"
	if !stdoutMermaid && !fileMermaid {
		buildIamrolelistOutput(roles, policies, s).Write()
		return
	}
	chart := iamrolelistMermaid(roles, policies, s.MermaidSettings)

	if stdoutMermaid {
		printIamrolelistBytes(chart, "", s.S3Bucket)
	} else {
		stdoutSettings := *s
		stdoutSettings.OutputFile = ""
		// Append applies to the file; html would otherwise read file "".
		stdoutSettings.ShouldAppend = false
		buildIamrolelistOutput(roles, policies, &stdoutSettings).Write()
	}

	switch {
	case s.OutputFile == "":
	case fileMermaid:
		printIamrolelistBytes(chart, s.OutputFile, s.S3Bucket)
	default:
		fileSettings := *s
		fileSettings.OutputFormat = fileFormat
		fileSettings.OutputFileFormat = fileFormat
		writeIamrolelistFileOnly(buildIamrolelistOutput(roles, policies, &fileSettings))
	}
}

// writeIamrolelistFileOnly writes only the file destination of output, in
// output.Settings.OutputFormat. go-output's Write always writes stdout as
// well, so stdout is discarded. Its html and drawio file branches write the
// file directly and then overwrite it with whatever the stdout branch
// produced, so html is built here (as go-output's toHTML does) and drawio
// dispatches stdout as drawio, which writes the same file.
func writeIamrolelistFileOnly(output format.OutputArray) {
	if output.Settings.OutputFormat == "html" {
		page, err := iamrolelistHTML(output)
		if err != nil {
			log.Fatal(err.Error())
		}
		printIamrolelistBytes(page, output.Settings.OutputFile, output.Settings.S3Bucket)
		return
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Fatal(err.Error())
	}
	orig := os.Stdout
	os.Stdout = devNull
	defer func() {
		os.Stdout = orig
		_ = devNull.Close()
	}()
	output.Write()
}

// iamrolelistHTML builds an HTML page like go-output's toHTML: the table is
// inserted at the end marker of the base template, or of the existing file
// when appending.
func iamrolelistHTML(output format.OutputArray) ([]byte, error) {
	const endMarker = "<div id='end'></div>"
	var base string
	if output.Settings.ShouldAppend {
		existing, err := os.ReadFile(output.Settings.OutputFile)
		if err != nil {
			return nil, err
		}
		base = string(existing)
	} else {
		tmpl, err := template.New("base").Parse(templates.BaseHTMLTemplate)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, output); err != nil {
			return nil, err
		}
		base = buf.String()
	}
	table := string(output.HtmlTableOnly()) + endMarker
	return []byte(strings.Replace(base, endMarker, table, 1)), nil
}

// buildIamrolelistOutput builds the rows for s.OutputFormat.
func buildIamrolelistOutput(roles []helpers.IAMRole, policies map[string]helpers.IAMPolicyDocument, s *format.OutputSettings) format.OutputArray {
	isDrawIO := s.OutputFormat == "drawio"
	// DOT uses the Name column as node identity.
	isDot := s.OutputFormat == "dot"
	keys := []string{nameColumn, typeColumn, "ARN", "AssumedFrom", "Policies", "Roles"}
	if isDrawIO {
		keys = append(keys, imageColumn, "DrawioID")
		s.DrawIOHeader = createIamrolelistDrawIOHeader()
	}
	output := format.OutputArray{Keys: keys, Settings: s}
	if s.NeedsFromToColumns() {
		s.AddFromToColumns(nameColumn, "Policies")
	}
	for _, role := range roles {
		content := make(map[string]any)
		content[nameColumn] = role.Name
		content["AssumedFrom"] = role.CanBeAssumedFrom()
		content[typeColumn] = role.Type
		content["Policies"] = role.GetPolicyNames()
		if isDot || isDrawIO {
			content["Policies"] = role.GetPolicyIDs()
		}
		if isDrawIO {
			content["DrawioID"] = role.ID
			content[imageColumn] = drawio.AWSShape("Security Identity Compliance", "Role")
		}
		output.AddHolder(format.OutputHolder{Contents: content})
	}
	for policyID, policy := range policies {
		content := make(map[string]any)
		content[nameColumn] = policyID
		if isDot {
			// Without a value go-output renders the missing link column as
			// "%!s(<nil>)" and adds an edge to a non-existent node.
			content["Policies"] = ""
		}
		if policy.Type == helpers.IAMPolicyTypeAttached {
			content["ARN"] = policyID
			if !isDot {
				content[nameColumn] = policy.Name
			}
		}
		if isDrawIO {
			content["DrawioID"] = policyID
			content[imageColumn] = drawio.AWSShape("Security Identity Compliance", "Permissions")
		}
		content[typeColumn] = policy.Type
		content["Roles"] = policy.GetRoleNames()
		output.AddHolder(format.OutputHolder{Contents: content})
	}
	return output
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
