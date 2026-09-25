package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// An IAM API failure should be returned for Execute to print once.
// Cobra must not also print the error or userlist usage.
func TestIAMUserListReturnsErrorWithoutPrintingUsage_T1549(t *testing.T) {
	wantErr := errors.New("list access keys for IAM user \"test-user\": access denied")
	userlist := *userlistCmd
	userlist.RunE = func(*cobra.Command, []string) error { return wantErr }
	iam := &cobra.Command{Use: "iam"}
	iam.AddCommand(&userlist)
	root := &cobra.Command{Use: "awstools"}
	root.AddCommand(iam)
	root.SetArgs([]string{"iam", "userlist"})
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)

	if err := root.Execute(); !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if output.Len() != 0 {
		t.Fatalf("Cobra printed %q; Execute is responsible for reporting the error once", output.String())
	}
}
