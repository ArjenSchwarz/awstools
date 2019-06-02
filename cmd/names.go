package cmd

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

var appendTo *bool

// namesCmd represents the names command
var namesCmd = &cobra.Command{
	Use:   "names",
	Short: "Get the names for the resources in the account",
	Long: `These names can be stored in a file and then used by other functionalities.
	This is especially useful for commands that deal with multiple accounts`,
	Run: names,
}

func init() {
	RootCmd.AddCommand(namesCmd)
}

func names(cmd *cobra.Command, args []string) {
	svc := helpers.Ec2Session()
	names := helpers.GetAllEC2ResourceNames(svc)
	if *settings.AppendToOutput {
		originalfile, err := ioutil.ReadFile(*settings.OutputFile)
		if err != nil {
			panic(err)
		}
		pulledin := make(map[string]string)
		err = json.Unmarshal(originalfile, &pulledin)
		if err != nil {
			panic(err)
		}
		for key, value := range names {
			pulledin[key] = value
		}
		names = pulledin
	}
	var file io.Writer
	var err error
	if *settings.OutputFile == "" {
		file = os.Stdout
	} else {
		file, err = os.Create(*settings.OutputFile)
		if err != nil {
			panic(err)
		}
	}
	responseString, _ := json.Marshal(names)
	w := bufio.NewWriter(file)
	w.Write(responseString)
	w.Flush()
}
