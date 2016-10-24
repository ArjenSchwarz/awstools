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
	Short: "Various IAM commands",
	Long:  `Various commands that deal with IAM users`,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	// TODO: Work your own magic here
	// 	fmt.Println("userlist called")
	// },
}

func init() {
	RootCmd.AddCommand(iamCmd)
}

//
// for _, function := range plugin.MainChecks {
// 	go func(function func() (slack.Attachment, error)) {
// 		attachment, err := function()
// 		if err != nil {
// 			// return response, err
// 		}
// 		c <- attachment
// 	}(function)
// }
// for i := 0; i < len(plugin.MainChecks); i++ {
// 	response.AddAttachment(<-c)
// }
