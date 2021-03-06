package cmd

import (
	"github.com/spf13/cobra"
)

// ssoCmd represents the tgw command
var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "AWS Single Sign-On commands",
	Long:  `Various AWS SSO commands`,
}

func init() {
	rootCmd.AddCommand(ssoCmd)
}

var ssoresourceid string
