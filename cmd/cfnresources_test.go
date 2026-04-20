package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// identityNameResolver is a name resolver that returns the id unchanged,
// mirroring the behaviour of getName when no namefile is configured.
func identityNameResolver(id string) string { return id }

// TestBuildCfnResource_NilPhysicalResourceId_T733 verifies that building a
// cfnResource from a StackResource whose PhysicalResourceId is nil does not
// panic and falls back to the logical resource id for the resource name.
//
// Bug (T-733): cmd/cfnresources.go dereferenced *resource.PhysicalResourceId
// without a nil guard, causing a panic for resources that haven't been created
// yet or for resource types that don't populate this field.
func TestBuildCfnResource_NilPhysicalResourceId_T733(t *testing.T) {
	resource := types.StackResource{
		StackName:          aws.String("my-stack"),
		LogicalResourceId:  aws.String("MyResource"),
		PhysicalResourceId: nil, // the condition that used to panic
		ResourceType:       aws.String("AWS::S3::Bucket"),
		ResourceStatus:     types.ResourceStatusCreateInProgress,
	}

	got := buildCfnResource(resource, identityNameResolver)

	if got.ResourceID != "" {
		t.Errorf("expected empty ResourceID when PhysicalResourceId is nil, got %q", got.ResourceID)
	}
	if got.ResourceName != "MyResource" {
		t.Errorf("expected ResourceName to fall back to LogicalResourceId %q, got %q", "MyResource", got.ResourceName)
	}
	if got.LogicalName != "MyResource" {
		t.Errorf("expected LogicalName %q, got %q", "MyResource", got.LogicalName)
	}
	if got.Stack != "my-stack" {
		t.Errorf("expected Stack %q, got %q", "my-stack", got.Stack)
	}
	if got.Type != "AWS::S3::Bucket" {
		t.Errorf("expected Type %q, got %q", "AWS::S3::Bucket", got.Type)
	}
}

// TestBuildCfnResource_PopulatedPhysicalResourceId_T733 verifies the happy path:
// when PhysicalResourceId is present it is used as ResourceID and passed to the
// name resolver.
func TestBuildCfnResource_PopulatedPhysicalResourceId_T733(t *testing.T) {
	resource := types.StackResource{
		StackName:          aws.String("my-stack"),
		LogicalResourceId:  aws.String("MyResource"),
		PhysicalResourceId: aws.String("my-resource-id-12345"),
		ResourceType:       aws.String("AWS::S3::Bucket"),
		ResourceStatus:     types.ResourceStatusCreateComplete,
	}

	resolver := func(id string) string {
		if id == "my-resource-id-12345" {
			return "FriendlyName"
		}
		return id
	}

	got := buildCfnResource(resource, resolver)

	if got.ResourceID != "my-resource-id-12345" {
		t.Errorf("expected ResourceID %q, got %q", "my-resource-id-12345", got.ResourceID)
	}
	if got.ResourceName != "FriendlyName" {
		t.Errorf("expected ResourceName %q, got %q", "FriendlyName", got.ResourceName)
	}
}

// TestBuildCfnResource_NilPhysicalResourceId_NilLogicalResourceId_T733 verifies
// that when both PhysicalResourceId and LogicalResourceId are nil, the function
// does not panic and produces empty identifiers.
func TestBuildCfnResource_NilPhysicalResourceId_NilLogicalResourceId_T733(t *testing.T) {
	resource := types.StackResource{
		StackName:          aws.String("my-stack"),
		LogicalResourceId:  nil,
		PhysicalResourceId: nil,
		ResourceType:       aws.String("AWS::S3::Bucket"),
	}

	got := buildCfnResource(resource, identityNameResolver)

	if got.ResourceID != "" {
		t.Errorf("expected empty ResourceID, got %q", got.ResourceID)
	}
	if got.ResourceName != "" {
		t.Errorf("expected empty ResourceName, got %q", got.ResourceName)
	}
	if got.LogicalName != "" {
		t.Errorf("expected empty LogicalName, got %q", got.LogicalName)
	}
}
