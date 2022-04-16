package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var settings = new(config.Config)
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "awstools",
	Short: "Various tools for dealing with complex AWS comments",
	Long: `awstools is designed to be used for more complex tasks that would take a lot of work using just the CLI.

This usually involves tasks that would require multiple calls.

Full documentation for all commands can be accessed using the --help flag or by reading it on https://arjenschwarz.github.io/awstools/
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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .awstools.yaml in current directory, or $HOME/.awstools.yaml)")

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	rootCmd.PersistentFlags().StringP("file", "f", "", "Optional file to save the output to")
	rootCmd.PersistentFlags().StringP("output", "o", "json", "Format for the output, currently supported are csv, table, json, html, dot, and drawio")
	rootCmd.PersistentFlags().BoolP("append", "a", false, "Add to the provided output file instead of replacing it")
	rootCmd.PersistentFlags().StringP("namefile", "n", "", "Use this file to provide names")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific profile")
	rootCmd.PersistentFlags().String("region", "", "Use a specific region")
	rootCmd.PersistentFlags().Bool("emoji", false, "Use emoji in the output")

	viper.BindPFlag("output.verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("aws.profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("aws.region", rootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("output.file", rootCmd.PersistentFlags().Lookup("file"))
	viper.BindPFlag("output.append", rootCmd.PersistentFlags().Lookup("append"))
	viper.BindPFlag("output.namefile", rootCmd.PersistentFlags().Lookup("namefile"))
	viper.BindPFlag("output.use-emoji", rootCmd.PersistentFlags().Lookup("emoji"))

	viper.SetDefault("output.table.style", "Default")
	viper.SetDefault("output.table.max-column-width", 50)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)
		// Default to local config file
		viper.AddConfigPath(".")
		// Search config in home directory with name ".awstools" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".awstools")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getName(id string) string {
	if settings.GetString("output.namefile") != "" {
		nameFile, err := ioutil.ReadFile(settings.GetString("output.namefile"))
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
