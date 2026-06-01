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

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// tablesCmd represents the tables command
var tablesCmd = &cobra.Command{
	Use:   "tables",
	Short: "Show what the different table styles look like",
	Long:  `This command will show an overview of all the different style of tables`,
	Run:   demoTables,
}

func init() {
	demoCmd.AddCommand(tablesCmd)
}

func demoTables(_ *cobra.Command, _ []string) {
	keys := []string{exportColumn, descriptionColumn, stackColumn, valueColumn, importedColumn}
	title := "CloudFormation Export values demo"
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = title
	output.Settings.SortKey = exportColumn

	value1 := format.OutputHolder{
		Contents: map[string]any{
			exportColumn:      "awesome-stack-dev-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-dev",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-dev",
			importedColumn:    true,
		},
	}
	value2 := format.OutputHolder{
		Contents: map[string]any{
			exportColumn:      "awesome-stack-test-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-test",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-test",
			importedColumn:    true,
		},
	}
	value3 := format.OutputHolder{
		Contents: map[string]any{
			exportColumn:      "awesome-stack-prod-s3-arn",
			valueColumn:       "arn:aws:s3:::fog-awesome-stack-prod",
			descriptionColumn: s3BucketARNDescription,
			stackColumn:       "awesome-stack-prod",
			importedColumn:    true,
		},
	}
	value4 := format.OutputHolder{
		Contents: map[string]any{
			exportColumn:      "demo-s3-bucket",
			valueColumn:       "fog-demo-bucket",
			descriptionColumn: "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
			stackColumn:       "demo-resources",
			importedColumn:    true,
		},
	}
	output.AddHolder(value1)
	output.AddHolder(value2)
	output.AddHolder(value3)
	output.AddHolder(value4)
	fmt.Print(`Tables can be used for the various outputs. You can set your preferred style in your settings file.
An example if you use .awstools.yaml as your settings file:

table:
  style: Default
  max-column-width: 50

`)
	for name, style := range format.TableStyles {
		output.Settings.TableStyle = style
		fmt.Println("")
		fmt.Printf("Showing style: %v\r\n", name)
		fmt.Println("")
		output.Write()
	}
}
