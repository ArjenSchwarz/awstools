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
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// structureCmd represents the structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Get a graphical overview of the Organization's structure",
	Long: `This gives you an overview of how the accounts are connected.
	Using the dot output format you can turn this into an image.

	Example: awstools organizations structure -o dot | dot -Tpng -o structure.png`,
	Run: orgstructure,
}

func init() {
	organizationsCmd.AddCommand(structureCmd)
}

func orgstructure(cmd *cobra.Command, args []string) {
	svc := helpers.OrganizationsSession()
	organization := helpers.GetFullOrganization(svc)
	keys := []string{"From", "To"}
	output := helpers.OutputArray{Keys: keys}
	traverseOrgStructureEntry(organization, &output)
	output.Write(*settings)
}

func traverseOrgStructureEntry(entry helpers.OrganizationEntry, output *helpers.OutputArray) {
	for _, child := range entry.Children {
		content := make(map[string]string)
		content["From"] = entry.String()
		content["To"] = child.String()
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
		traverseOrgStructureEntry(child, output)
	}
	if len(entry.Children) == 0 {
		content := make(map[string]string)
		content["From"] = entry.String()
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
}
