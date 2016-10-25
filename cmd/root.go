package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var settings = new(config.Config)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "awstools",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	// cobra.OnInitialize(initConfig)
	settings.Verbose = RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	settings.OutputFile = RootCmd.PersistentFlags().StringP("output", "o", "", "Optional file to save the output to")
	settings.OutputFormat = RootCmd.PersistentFlags().StringP("format", "f", "csv", "Format for the output, currently supported are csv and json")
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
