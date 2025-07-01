package helpers

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestGetRoleType(t *testing.T) {
	tests := []struct {
		name     string
		role     types.Role
		expected string
	}{
		{
			name: "service role path",
			role: types.Role{
				Path: aws.String("/service-role/"),
			},
			expected: IAMRoleTypeServiceRole,
		},
		{
			name: "aws service role path",
			role: types.Role{
				Path: aws.String("/aws-service-role/elasticloadbalancing.amazonaws.com/"),
			},
			expected: IAMRoleTypeServiceRole,
		},
		{
			name: "sso managed role path",
			role: types.Role{
				Path: aws.String("/aws-reserved/sso.amazonaws.com/us-west-2_123456789012/"),
			},
			expected: IAMRoleTypeSSOManaged,
		},
		{
			name: "user defined role default path",
			role: types.Role{
				Path: aws.String("/"),
			},
			expected: IAMRoleTypeUserDefined,
		},
		{
			name: "user defined role custom path",
			role: types.Role{
				Path: aws.String("/my-custom-path/"),
			},
			expected: IAMRoleTypeUserDefined,
		},
		{
			name: "edge case - short aws-service-role path",
			role: types.Role{
				Path: aws.String("/aws-service-role/"),
			},
			expected: IAMRoleTypeUserDefined, // Only paths with length > 18 and starting with /aws-service-role/ are service roles
		},
		{
			name: "edge case - short sso path",
			role: types.Role{
				Path: aws.String("/aws-reserved/sso.amazonaws.com/"),
			},
			expected: IAMRoleTypeUserDefined, // Less than 32 characters, so not SSO
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRoleType(tt.role)
			if result != tt.expected {
				t.Errorf("getRoleType() = %s, want %s for path %s", result, tt.expected, aws.ToString(tt.role.Path))
			}
		})
	}
}

func TestIAMRole_CanBeAssumedFrom_ExtendedCases(t *testing.T) {
	tests := []struct {
		name     string
		role     IAMRole
		expected []string
	}{
		{
			name: "role with multiple statements",
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
						{
							Effect: "Allow",
							Action: "sts:AssumeRoleWithSAML",
							Principal: map[string]string{
								"Federated": "arn:aws:iam::123456789012:saml-provider/Provider",
							},
						},
					},
				},
			},
			expected: []string{"Service: ec2.amazonaws.com", "Federated: arn:aws:iam::123456789012:saml-provider/Provider"},
		},
		{
			name: "role with deny effect - function doesn't check Effect field",
			role: IAMRole{
				AssumeRolePolicy: IAMPolicyDocument{
					Statement: []IAMPolicyDocumentStatement{
						{
							Effect: "Deny",
							Action: "sts:AssumeRole",
							Principal: map[string]string{
								"Service": "ec2.amazonaws.com",
							},
						},
					},
				},
			},
			expected: []string{"Service: ec2.amazonaws.com"}, // Function currently doesn't check Effect
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

func TestIAMRole_GetPolicyNames_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		role     IAMRole
		expected int
	}{
		{
			name: "role with no policies",
			role: IAMRole{
				Name:             "empty-role",
				InlinePolicies:   map[string]*IAMPolicyDocument{},
				AttachedPolicies: map[string]*IAMPolicyDocument{},
			},
			expected: 0,
		},
		{
			name: "role with only inline policies",
			role: IAMRole{
				Name: "inline-only-role",
				InlinePolicies: map[string]*IAMPolicyDocument{
					"inline1": {Name: "inline1"},
					"inline2": {Name: "inline2"},
				},
				AttachedPolicies: map[string]*IAMPolicyDocument{},
			},
			expected: 2,
		},
		{
			name: "role with only attached policies",
			role: IAMRole{
				Name:           "attached-only-role",
				InlinePolicies: map[string]*IAMPolicyDocument{},
				AttachedPolicies: map[string]*IAMPolicyDocument{
					"AmazonS3ReadOnlyAccess": {Name: "AmazonS3ReadOnlyAccess"},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.GetPolicyNames()
			if len(result) != tt.expected {
				t.Errorf("GetPolicyNames() returned %d policies, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestCachedIAMPolicyDocuments(t *testing.T) {
	// Test that the cache is initialized as a map
	if cachedIAMPolicyDocuments == nil {
		t.Errorf("cachedIAMPolicyDocuments should be initialized")
	}

	// Test we can add to the cache
	testPolicy := &IAMPolicyDocument{
		Name: "test-policy",
		Type: IAMPolicyTypeInline,
	}

	cachedIAMPolicyDocuments["test-key"] = testPolicy

	if len(cachedIAMPolicyDocuments) == 0 {
		t.Errorf("Expected cachedIAMPolicyDocuments to contain test policy")
	}

	retrievedPolicy, exists := cachedIAMPolicyDocuments["test-key"]
	if !exists {
		t.Errorf("Expected to find test-key in cache")
	}

	if retrievedPolicy.Name != "test-policy" {
		t.Errorf("Expected cached policy name to be 'test-policy', got %s", retrievedPolicy.Name)
	}

	// Clean up the test
	delete(cachedIAMPolicyDocuments, "test-key")
}

// Integration tests would require IAM access
func TestGetRolesAndPolicies_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires IAM client interface implementation")
}

func TestGetRoleDetails_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires IAM client interface implementation")
}

func TestGetInlinePoliciesForRole_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires IAM client interface implementation")
}

func TestGetAttachedPoliciesForRole_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires IAM client interface implementation")
}
