package helpers

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestIAMUser_GetAllPolicies(t *testing.T) {
	user := IAMUser{
		Name: "test-user",
		InlinePolicies: map[string]string{
			"inline-policy-1": `{"Version":"2012-10-17"}`,
			"inline-policy-2": `{"Version":"2012-10-17"}`,
		},
		AttachedPolicies: map[string]string{
			"attached-policy-1": `{"Version":"2012-10-17"}`,
		},
		InlineGroupPolicies: map[string]string{
			"group-inline-policy": `{"Version":"2012-10-17"}`,
		},
		AttachedGroupPolicies: map[string]string{
			"group-attached-policy": `{"Version":"2012-10-17"}`,
		},
	}

	expected := map[string]string{
		"inline-policy-1":       `{"Version":"2012-10-17"}`,
		"inline-policy-2":       `{"Version":"2012-10-17"}`,
		"attached-policy-1":     `{"Version":"2012-10-17"}`,
		"group-inline-policy":   `{"Version":"2012-10-17"}`,
		"group-attached-policy": `{"Version":"2012-10-17"}`,
	}

	result := user.GetAllPolicies()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GetAllPolicies() = %v, want %v", result, expected)
	}
}

func TestIAMUser_Interface(t *testing.T) {
	user := IAMUser{
		Name:   "test-user",
		ID:     "AIDACKCEVSQ6C2EXAMPLE",
		Groups: []string{"group1", "group2"},
		InlinePolicies: map[string]string{
			"inline-policy": `{"Version":"2012-10-17"}`,
		},
		AttachedPolicies: map[string]string{
			"attached-policy": `{"Version":"2012-10-17"}`,
		},
		InlineGroupPolicies: map[string]string{
			"group-inline": `{"Version":"2012-10-17"}`,
		},
		AttachedGroupPolicies: map[string]string{
			"group-attached": `{"Version":"2012-10-17"}`,
		},
	}

	// Test IAMObject interface methods
	if user.GetName() != "test-user" {
		t.Errorf("GetName() = %s, want test-user", user.GetName())
	}

	if user.GetID() != "AIDACKCEVSQ6C2EXAMPLE" {
		t.Errorf("GetID() = %s, want AIDACKCEVSQ6C2EXAMPLE", user.GetID())
	}

	if len(user.GetUsers()) != 0 {
		t.Errorf("GetUsers() should return empty slice for user, got %v", user.GetUsers())
	}

	expectedGroups := []string{"group1", "group2"}
	if !reflect.DeepEqual(user.GetGroups(), expectedGroups) {
		t.Errorf("GetGroups() = %v, want %v", user.GetGroups(), expectedGroups)
	}

	if user.GetObjectType() != IAMObjectTypeUser {
		t.Errorf("GetObjectType() = %s, want %s", user.GetObjectType(), IAMObjectTypeUser)
	}

	expectedDirectPolicies := map[string]string{
		"inline-policy":   `{"Version":"2012-10-17"}`,
		"attached-policy": `{"Version":"2012-10-17"}`,
	}
	if !reflect.DeepEqual(user.GetDirectPolicies(), expectedDirectPolicies) {
		t.Errorf("GetDirectPolicies() = %v, want %v", user.GetDirectPolicies(), expectedDirectPolicies)
	}

	expectedInheritedPolicies := map[string]string{
		"group-inline":   `{"Version":"2012-10-17"}`,
		"group-attached": `{"Version":"2012-10-17"}`,
	}
	if !reflect.DeepEqual(user.GetInheritedPolicies(), expectedInheritedPolicies) {
		t.Errorf("GetInheritedPolicies() = %v, want %v", user.GetInheritedPolicies(), expectedInheritedPolicies)
	}
}

func TestIAMUser_PasswordAndAccessKeys(t *testing.T) {
	// Test user with password last used
	testTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	userWithPassword := IAMUser{
		User: &types.User{
			PasswordLastUsed: &testTime,
		},
	}

	if !userWithPassword.HasUsedPassword() {
		t.Errorf("HasUsedPassword() should return true when PasswordLastUsed is set")
	}

	// Test user without password
	userWithoutPassword := IAMUser{
		User: &types.User{
			PasswordLastUsed: nil,
		},
	}

	if userWithoutPassword.HasUsedPassword() {
		t.Errorf("HasUsedPassword() should return false when PasswordLastUsed is nil")
	}
}

func TestIAMGroup_Interface(t *testing.T) {
	group := IAMGroup{
		Name:  "test-group",
		ID:    "AGPAI23HZ27SI6FQMGNQ2",
		Users: []string{"user1", "user2"},
		InlinePolicies: map[string]string{
			"inline-policy": `{"Version":"2012-10-17"}`,
		},
		AttachedPolicies: map[string]string{
			"attached-policy": `{"Version":"2012-10-17"}`,
		},
	}

	// Test IAMObject interface methods
	if group.GetName() != "test-group" {
		t.Errorf("GetName() = %s, want test-group", group.GetName())
	}

	if group.GetID() != "AGPAI23HZ27SI6FQMGNQ2" {
		t.Errorf("GetID() = %s, want AGPAI23HZ27SI6FQMGNQ2", group.GetID())
	}

	expectedUsers := []string{"user1", "user2"}
	if !reflect.DeepEqual(group.GetUsers(), expectedUsers) {
		t.Errorf("GetUsers() = %v, want %v", group.GetUsers(), expectedUsers)
	}

	if len(group.GetGroups()) != 0 {
		t.Errorf("GetGroups() should return empty slice for group, got %v", group.GetGroups())
	}

	if group.GetObjectType() != IAMObjectTypeGroup {
		t.Errorf("GetObjectType() = %s, want %s", group.GetObjectType(), IAMObjectTypeGroup)
	}

	expectedDirectPolicies := map[string]string{
		"inline-policy":   `{"Version":"2012-10-17"}`,
		"attached-policy": `{"Version":"2012-10-17"}`,
	}
	if !reflect.DeepEqual(group.GetDirectPolicies(), expectedDirectPolicies) {
		t.Errorf("GetDirectPolicies() = %v, want %v", group.GetDirectPolicies(), expectedDirectPolicies)
	}

	// Groups have no inherited policies
	if len(group.GetInheritedPolicies()) != 0 {
		t.Errorf("GetInheritedPolicies() should return empty map for group, got %v", group.GetInheritedPolicies())
	}
}

func TestAttachedIAMPolicy_AddObject(t *testing.T) {
	policy := AttachedIAMPolicy{
		Name:   "test-policy",
		Users:  []string{},
		Groups: []string{},
	}

	// Add a user
	user := IAMUser{
		Name: "test-user",
	}
	policy.AddObject(user)

	expectedUsers := []string{"test-user"}
	if !reflect.DeepEqual(policy.Users, expectedUsers) {
		t.Errorf("After adding user, Users = %v, want %v", policy.Users, expectedUsers)
	}

	// Add a group
	group := IAMGroup{
		Name: "test-group",
	}
	policy.AddObject(group)

	expectedGroups := []string{"test-group"}
	if !reflect.DeepEqual(policy.Groups, expectedGroups) {
		t.Errorf("After adding group, Groups = %v, want %v", policy.Groups, expectedGroups)
	}

	// Final state check
	if len(policy.Users) != 1 || policy.Users[0] != "test-user" {
		t.Errorf("Users should contain 'test-user', got %v", policy.Users)
	}
	if len(policy.Groups) != 1 || policy.Groups[0] != "test-group" {
		t.Errorf("Groups should contain 'test-group', got %v", policy.Groups)
	}
}

func TestIAMConstants(t *testing.T) {
	// Test IAM Role Type constants
	if IAMRoleTypeSSOManaged != "SSO Managed Role" {
		t.Errorf("IAMRoleTypeSSOManaged = %s, want 'SSO Managed Role'", IAMRoleTypeSSOManaged)
	}
	if IAMRoleTypeServiceRole != "Service Role" {
		t.Errorf("IAMRoleTypeServiceRole = %s, want 'Service Role'", IAMRoleTypeServiceRole)
	}
	if IAMRoleTypeUserDefined != "User defined Role" {
		t.Errorf("IAMRoleTypeUserDefined = %s, want 'User defined Role'", IAMRoleTypeUserDefined)
	}

	// Test IAM Policy Type constants
	if IAMPolicyTypeAttached != "Attached Policy" {
		t.Errorf("IAMPolicyTypeAttached = %s, want 'Attached Policy'", IAMPolicyTypeAttached)
	}
	if IAMPolicyTypeInline != "Inline Policy" {
		t.Errorf("IAMPolicyTypeInline = %s, want 'Inline Policy'", IAMPolicyTypeInline)
	}
	if IAMPolicyTypeAssumeRole != "Assume Role Policy" {
		t.Errorf("IAMPolicyTypeAssumeRole = %s, want 'Assume Role Policy'", IAMPolicyTypeAssumeRole)
	}

	// Test IAM Object Type constants
	if IAMObjectTypeGroup != "Group" {
		t.Errorf("IAMObjectTypeGroup = %s, want 'Group'", IAMObjectTypeGroup)
	}
	if IAMObjectTypeUser != "User" {
		t.Errorf("IAMObjectTypeUser = %s, want 'User'", IAMObjectTypeUser)
	}
}

func TestIAMRole_CanBeAssumedFrom(t *testing.T) {
	tests := []struct {
		name     string
		role     IAMRole
		expected []string
	}{
		{
			name: "role with EC2 service principal",
			role: IAMRole{
				AssumeRolePolicy: IAMPolicyDocument{
					Statement: []IAMPolicyDocumentStatement{
						{
							Effect: "Allow",
							Action: "sts:AssumeRole",
							Principal: map[string]string{
								"Service": "ec2.amazonaws.com",
							},
						},
					},
				},
			},
			expected: []string{"Service: ec2.amazonaws.com"},
		},
		{
			name: "role with multiple principals",
			role: IAMRole{
				AssumeRolePolicy: IAMPolicyDocument{
					Statement: []IAMPolicyDocumentStatement{
						{
							Effect: "Allow",
							Action: "sts:AssumeRole",
							Principal: map[string]string{
								"Service": "lambda.amazonaws.com",
								"AWS":     "arn:aws:iam::123456789012:root",
							},
						},
					},
				},
			},
			expected: []string{"AWS: arn:aws:iam::123456789012:root", "Service: lambda.amazonaws.com"},
		},
		{
			name: "role with SAML assumption",
			role: IAMRole{
				AssumeRolePolicy: IAMPolicyDocument{
					Statement: []IAMPolicyDocumentStatement{
						{
							Effect: "Allow",
							Action: "sts:AssumeRoleWithSAML",
							Principal: map[string]string{
								"Federated": "arn:aws:iam::123456789012:saml-provider/ExampleProvider",
							},
						},
					},
				},
			},
			expected: []string{"Federated: arn:aws:iam::123456789012:saml-provider/ExampleProvider"},
		},
		{
			name: "role with no matching actions",
			role: IAMRole{
				AssumeRolePolicy: IAMPolicyDocument{
					Statement: []IAMPolicyDocumentStatement{
						{
							Effect: "Allow",
							Action: "s3:GetObject",
							Principal: map[string]string{
								"Service": "ec2.amazonaws.com",
							},
						},
					},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.CanBeAssumedFrom()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CanBeAssumedFrom() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIAMRole_GetPolicyNames(t *testing.T) {
	role := IAMRole{
		Name: "test-role",
		InlinePolicies: map[string]*IAMPolicyDocument{
			"inline-policy-1": {Name: "inline-policy-1"},
			"inline-policy-2": {Name: "inline-policy-2"},
		},
		AttachedPolicies: map[string]*IAMPolicyDocument{
			"AmazonS3ReadOnlyAccess": {Name: "AmazonS3ReadOnlyAccess"},
			"CustomAttachedPolicy":   {Name: "CustomAttachedPolicy"},
		},
	}

	result := role.GetPolicyNames()

	// Check that we have the right number of policies
	if len(result) != 4 {
		t.Errorf("Expected 4 policy names, got %d", len(result))
	}

	// Check inline policies (should have role name suffix)
	foundInline1 := false
	foundInline2 := false
	for _, policyName := range result {
		if policyName == "inline-policy-1 (inline for test-role)" {
			foundInline1 = true
		}
		if policyName == "inline-policy-2 (inline for test-role)" {
			foundInline2 = true
		}
	}
	if !foundInline1 {
		t.Errorf("Expected to find 'inline-policy-1 (inline for test-role)' in result")
	}
	if !foundInline2 {
		t.Errorf("Expected to find 'inline-policy-2 (inline for test-role)' in result")
	}

	// Check attached policies (should not have suffix)
	foundAttached1 := false
	foundAttached2 := false
	for _, policyName := range result {
		if policyName == "AmazonS3ReadOnlyAccess" {
			foundAttached1 = true
		}
		if policyName == "CustomAttachedPolicy" {
			foundAttached2 = true
		}
	}
	if !foundAttached1 {
		t.Errorf("Expected to find 'AmazonS3ReadOnlyAccess' in result")
	}
	if !foundAttached2 {
		t.Errorf("Expected to find 'CustomAttachedPolicy' in result")
	}
}

func TestIAMPolicyDocument_AddRole(t *testing.T) {
	policy := IAMPolicyDocument{
		Name:  "test-policy",
		Roles: []*IAMRole{},
	}

	role1 := IAMRole{Name: "role1"}
	role2 := IAMRole{Name: "role2"}

	policy.AddRole(&role1)
	policy.AddRole(&role2)

	if len(policy.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(policy.Roles))
	}

	if policy.Roles[0].Name != "role1" {
		t.Errorf("Expected first role to be 'role1', got '%s'", policy.Roles[0].Name)
	}

	if policy.Roles[1].Name != "role2" {
		t.Errorf("Expected second role to be 'role2', got '%s'", policy.Roles[1].Name)
	}
}

func TestIAMPolicyDocument_GetRoleNames(t *testing.T) {
	policy := IAMPolicyDocument{
		Name: "test-policy",
		Roles: []*IAMRole{
			{Name: "role1"},
			{Name: "role2"},
			{Name: "role3"},
		},
	}

	result := policy.GetRoleNames()
	expected := []string{"role1", "role2", "role3"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GetRoleNames() = %v, want %v", result, expected)
	}
}

func TestIAMPolicyDocument_EmptyRoles(t *testing.T) {
	policy := IAMPolicyDocument{
		Name:  "test-policy",
		Roles: []*IAMRole{},
	}

	result := policy.GetRoleNames()
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}
