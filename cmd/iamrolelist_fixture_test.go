package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
)

const (
	awsReadOnlyArn      = "arn:aws:iam::aws:policy/ReadOnlyAccess"
	customerReadOnlyArn = "arn:aws:iam::123456789012:policy/ReadOnlyAccess"
	roleAInlineID       = "inline-a (inline for role-a)"
)

// sameNamedRolelistFixture mirrors GetRolesAndPolicies output for two roles
// attaching different managed policies named ReadOnlyAccess (T-1656). With
// withInline, role-a also has an inline policy.
func sameNamedRolelistFixture(withInline bool) ([]helpers.IAMRole, map[string]helpers.IAMPolicyDocument) {
	awsDoc := &helpers.IAMPolicyDocument{Name: "ReadOnlyAccess", Type: helpers.IAMPolicyTypeAttached}
	customerDoc := &helpers.IAMPolicyDocument{Name: "ReadOnlyAccess", Type: helpers.IAMPolicyTypeAttached}
	inlineDoc := &helpers.IAMPolicyDocument{Name: "inline-a", Type: helpers.IAMPolicyTypeInline}
	roles := []helpers.IAMRole{
		{
			Name: "role-a", ID: "AROAA", Type: helpers.IAMRoleTypeUserDefined,
			InlinePolicies:   map[string]*helpers.IAMPolicyDocument{"inline-a": inlineDoc},
			AttachedPolicies: map[string]*helpers.IAMPolicyDocument{awsReadOnlyArn: awsDoc},
		},
		{
			Name: "role-b", ID: "AROAB", Type: helpers.IAMRoleTypeUserDefined,
			InlinePolicies:   map[string]*helpers.IAMPolicyDocument{},
			AttachedPolicies: map[string]*helpers.IAMPolicyDocument{customerReadOnlyArn: customerDoc},
		},
	}
	awsDoc.AddRole(&roles[0])
	customerDoc.AddRole(&roles[1])
	policies := map[string]helpers.IAMPolicyDocument{
		awsReadOnlyArn:      *awsDoc,
		customerReadOnlyArn: *customerDoc,
	}
	if withInline {
		inlineDoc.AddRole(&roles[0])
		policies[roleAInlineID] = *inlineDoc
	} else {
		roles[0].InlinePolicies = map[string]*helpers.IAMPolicyDocument{}
	}
	return roles, policies
}

// captureStdout runs f while redirecting os.Stdout and returns what it wrote.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()
	f()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}
