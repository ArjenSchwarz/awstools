package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var settings = new(config.Config)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "awstools",
	Short: "Various tools for dealing with complex AWS comments",
	Long: `awstools is designed to be used for more complex tasks that would take a lot of work using just the CLI.

This usually involves tasks that would require multiple calls.

Full documentation for all commands can be accessed using the --help flag or by reading it on https://github.com/ArjenSchwarz/awstools/blob/main/docs/awstools.md
`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	// cobra.OnInitialize(initConfig)
	settings.Verbose = rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	settings.OutputFile = rootCmd.PersistentFlags().StringP("file", "f", "", "Optional file to save the output to")
	settings.OutputFormat = rootCmd.PersistentFlags().StringP("output", "o", "json", "Format for the output, currently supported are csv, json, html, dot, and drawio")
	settings.AppendToOutput = rootCmd.PersistentFlags().BoolP("append", "a", false, "Add to the provided output file instead of replacing it")
	settings.NameFile = rootCmd.PersistentFlags().StringP("namefile", "n", "", "Use this file to provide names")
	settings.Profile = rootCmd.PersistentFlags().String("profile", "", "Use a specific profile")
	settings.Region = rootCmd.PersistentFlags().String("region", "", "Use a specific region")
	settings.UseEmoji = rootCmd.PersistentFlags().Bool("emoji", false, "Use emoji in the output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName(".awstools") // name of config file (without extension)
	viper.AddConfigPath("$HOME")     // adding home directory as first search path
	viper.AutomaticEnv()             // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func getName(id string) string {
	if *settings.NameFile != "" {
		nameFile, err := ioutil.ReadFile(*settings.NameFile)
		if err != nil {
			panic(err)
		}
		values := make(map[string]string)
		err = json.Unmarshal(nameFile, &values)
		if err != nil {
			panic(err)
		}
		if val, ok := values[id]; ok {
			return val
		}
	}
	return id
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
