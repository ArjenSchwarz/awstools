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
package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// demosettingsCmd represents the settings command
var demosettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Information about the settings file",
	Long:  `This will show details about what settings are available in the settings file`,
	Run:   demosettings,
}

func init() {
	demoCmd.AddCommand(demosettingsCmd)
}

func demosettings(cmd *cobra.Command, args []string) {
	fmt.Print(`While you can provide many settings as command-line flags, for some settings it makes more sense to use a settings file.

Fog provides the option to create a settings file in different formats: YAML, JSON, or TOML and you can always override its values with a flag when you run your command.
You can do this by creating a file called fog.yaml (or .json/.toml) in either the project directory from which you run the command, or in your home directory.
The project directory will take precedence if both are present. In addition, using the --config flag you can provide a custom path for your settings file.

So, what are your current settings and what would they look like in a settings file?

`)
	sorted := viper.AllKeys()
	sort.Strings(sorted)
	// lastsection := ""
	// fmt.Print(viper.AllSettings())
	yamlconfig, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		panic(err)
	}
	jsonconfig, err := json.MarshalIndent(viper.AllSettings(), "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println("As .awstools.yml:")
	fmt.Println(string(yamlconfig))
	fmt.Println("As .awstools.json:")
	fmt.Println(string(jsonconfig))
}
