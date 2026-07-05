/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package cmd provides command-line interface implementations for awstools
package cmd

import (
	"fmt"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// tablesCmd represents the tables command
var tablesCmd = &cobra.Command{
	Use:   "tables",
	Short: "Show what the different table styles look like",
	Long:  `This command will show an overview of all the different style of tables`,
	RunE:  demoTables,
}

// demoTableStyles hardcodes the table style names v1 exposed through its
// format.TableStyles map. v2 keeps its style lookup unexported, so the demo
// iterates this list instead; TestDemoTableStylesAcceptedByV2 guards the list
// against drifting from what v2 accepts.
var demoTableStyles = []string{
	"Default",
	"Bold",
	"ColoredBright",
	"ColoredDark",
	"ColoredBlackOnBlueWhite",
	"ColoredBlackOnCyanWhite",
	"ColoredBlackOnGreenWhite",
	"ColoredBlackOnMagentaWhite",
	"ColoredBlackOnYellowWhite",
	"ColoredBlackOnRedWhite",
	"ColoredBlueWhiteOnBlack",
	"ColoredCyanWhiteOnBlack",
	"ColoredGreenWhiteOnBlack",
	"ColoredMagentaWhiteOnBlack",
	"ColoredRedWhiteOnBlack",
	"ColoredYellowWhiteOnBlack",
}

func init() {
	demoCmd.AddCommand(tablesCmd)
}

// demoTableDocument builds the single-table demo document rendered once per
// table style. Rows are pre-sorted by the Export column, matching the
// SortKey behaviour of the v1 implementation.
func demoTableDocument() *output.Document {
	keys := []string{exportColumn, descriptionColumn, stackColumn, valueColumn, importedColumn}
	rows := []map[string]any{
		{
			exportColumn:      "awesome-stack-dev-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-dev",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-dev",
			importedColumn:    true,
		},
		{
			exportColumn:      "awesome-stack-prod-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-prod",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-prod",
			importedColumn:    true,
		},
		{
			exportColumn:      "awesome-stack-test-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-test",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-test",
			importedColumn:    true,
		},
		{
			exportColumn:      "demo-s3-bucket",
			valueColumn:       "fog-demo-bucket",
			descriptionColumn: "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
			stackColumn:       "demo-resources",
			importedColumn:    true,
		},
	}
	return output.New().
		Table("CloudFormation Export values demo", rows, output.WithKeys(keys...)).
		Build()
}

// demoTables renders the demo document once per supported table style. It
// builds an inline Output per iteration because the style deliberately varies;
// --file support is intentionally not offered for this demo command.
func demoTables(cmd *cobra.Command, _ []string) error {
	doc := demoTableDocument()
	fmt.Print(`Tables can be used for the various outputs. You can set your preferred style in your settings file.
An example if you use .awstools.yaml as your settings file:

table:
  style: Default
  max-column-width: 50

`)
	for _, name := range demoTableStyles {
		fmt.Println("")
		fmt.Printf("Showing style: %v\r\n", name)
		fmt.Println("")
		out := output.NewOutput(
			output.WithFormat(output.TableWithStyle(name)),
			output.WithWriter(output.NewStdoutWriter()),
		)
		if err := out.Render(cmd.Context(), doc); err != nil {
			return err
		}
	}
	return nil
}
