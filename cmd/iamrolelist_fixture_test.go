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
	securityAuditArn    = "arn:aws:iam::aws:policy/SecurityAudit"
	roleAInlineID       = "inline-a (inline for role-a)"
)

// sameNamedRolelistFixture mirrors GetRolesAndPolicies output for two roles
// attaching different managed policies named ReadOnlyAccess (T-1656). Both
// roles also attach SecurityAudit, and role-a has an inline policy, so each
// role has several policies and one policy has several roles.
func sameNamedRolelistFixture() ([]helpers.IAMRole, map[string]helpers.IAMPolicyDocument) {
	awsDoc := &helpers.IAMPolicyDocument{Name: "ReadOnlyAccess", Type: helpers.IAMPolicyTypeAttached}
	customerDoc := &helpers.IAMPolicyDocument{Name: "ReadOnlyAccess", Type: helpers.IAMPolicyTypeAttached}
	sharedDoc := &helpers.IAMPolicyDocument{Name: "SecurityAudit", Type: helpers.IAMPolicyTypeAttached}
	inlineDoc := &helpers.IAMPolicyDocument{Name: "inline-a", Type: helpers.IAMPolicyTypeInline}
	roles := []helpers.IAMRole{
		{
			Name: "role-a", ID: "AROAA", Type: helpers.IAMRoleTypeUserDefined,
			InlinePolicies:   map[string]*helpers.IAMPolicyDocument{"inline-a": inlineDoc},
			AttachedPolicies: map[string]*helpers.IAMPolicyDocument{awsReadOnlyArn: awsDoc, securityAuditArn: sharedDoc},
		},
		{
			Name: "role-b", ID: "AROAB", Type: helpers.IAMRoleTypeUserDefined,
			InlinePolicies:   map[string]*helpers.IAMPolicyDocument{},
			AttachedPolicies: map[string]*helpers.IAMPolicyDocument{customerReadOnlyArn: customerDoc, securityAuditArn: sharedDoc},
		},
	}
	inlineDoc.AddRole(&roles[0])
	awsDoc.AddRole(&roles[0])
	sharedDoc.AddRole(&roles[0])
	customerDoc.AddRole(&roles[1])
	sharedDoc.AddRole(&roles[1])
	policies := map[string]helpers.IAMPolicyDocument{
		awsReadOnlyArn:      *awsDoc,
		customerReadOnlyArn: *customerDoc,
		securityAuditArn:    *sharedDoc,
		roleAInlineID:       *inlineDoc,
	}
	return roles, policies
}

// captureIAMRoleListStdout runs f while redirecting os.Stdout and returns what it wrote.
func captureIAMRoleListStdout(t *testing.T, f func()) string {
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
