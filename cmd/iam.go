package cmd

import "github.com/spf13/cobra"

type iamUser struct {
	Username              string
	AttachedPolicies      map[string]string
	InlinePolicies        map[string]string
	Groups                []string
	AttachedGroupPolicies map[string]string
	InlineGroupPolicies   map[string]string
}

// iamCmd represents the iam command
var iamCmd = &cobra.Command{
	Use:   "iam",
	Short: "IAM commands",
	Long:  `Various commands that deal with IAM users`,
}

func init() {
	RootCmd.AddCommand(iamCmd)
}
