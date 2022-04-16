package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docsCmd represents the cfn command
var docsmarkdownCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate Markdown documentation for awstools",
	Long: `Generate documentation for awstools in Markdown format
This is used for the documentation in the repository, but can be run separately. By default it will generate it in the docs directory from where you run the command, but you can override this with the --directory flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := doc.GenMarkdownTree(rootCmd, docsdir)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	genCmd.AddCommand(docsmarkdownCmd)
	docsmarkdownCmd.Flags().StringVarP(&docsdir, "directory", "d", "./docs", "The directory where the documentation will be generated")

}
