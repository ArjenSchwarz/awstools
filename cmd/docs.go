package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docsCmd represents the cfn command
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation for awstools",
	Long: `Generate documentation for awstools in Markdown format
This is used for the documentation in the repository, but can be run separately. By default it will generate it in the docs directory from where you run the command, but you can override this with the --dir flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := doc.GenMarkdownTree(rootCmd, docsdir)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var docsdir string

func init() {
	genCmd.AddCommand(docsCmd)
	docsCmd.Flags().StringVarP(&docsdir, "directory", "d", "./docs", "The directory where the documentation will be generated")

}
