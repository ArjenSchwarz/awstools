package cmd

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docsCmd represents the documentation command
var docsghpagesCmd = &cobra.Command{
	Use:   "ghpages",
	Short: "Generate documentation for awstools' gh pages site",
	Long: `Generate documentation for awstools in Markdown format
This is used for the documentation in the GitHub Pages site, but can be run separately.`,
	Run: func(cmd *cobra.Command, args []string) {
		docsdir = "docs/content/commands"

		const fmTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---
`

		filePrepender := func(filename string) string {
			now := time.Now().Format(time.RFC3339)
			name := filepath.Base(filename)
			base := strings.TrimSuffix(name, path.Ext(name))
			url := "/awstools/" + strings.ToLower(base) + "/"
			return fmt.Sprintf(fmTemplate, now, strings.ReplaceAll(base, "_", " "), base, url)
		}
		linkHandler := func(name string) string {
			base := strings.TrimSuffix(name, path.Ext(name))
			return "#" + strings.ReplaceAll(strings.ToLower(base), "_", "-")
		}
		rootCmd.DisableAutoGenTag = true
		err := doc.GenMarkdownTreeCustom(rootCmd, docsdir, filePrepender, linkHandler)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	genCmd.AddCommand(docsghpagesCmd)
}
