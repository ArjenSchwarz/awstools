package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockDescribeNetworkInterfacesAPIClient implements the
// ec2.DescribeNetworkInterfacesAPIClient interface and paginates the provided
// interfaces using a fixed page size so tests can force multi-page responses.
type mockDescribeNetworkInterfacesAPIClient struct {
	interfaces []types.NetworkInterface
	pageSize   int
	callCount  int
}

// DescribeNetworkInterfaces returns a slice of the mock's interfaces, using
// NextToken to advance through pages. When no further pages exist, NextToken
// is left nil so the paginator stops.
func (m *mockDescribeNetworkInterfacesAPIClient) DescribeNetworkInterfaces(_ context.Context, input *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		fmt.Sscanf(*input.NextToken, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.interfaces)
	}
	end := start + pageSize
	if end > len(m.interfaces) {
		end = len(m.interfaces)
	}
	out := &ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: m.interfaces[start:end],
	}
	if end < len(m.interfaces) {
		token := fmt.Sprintf("%d", end)
		out.NextToken = &token
	}
	return out, nil
}

// makeNetworkInterfaces generates n synthetic ENI records with predictable IDs
// so tests can assert completeness across pages.
func makeNetworkInterfaces(n int) []types.NetworkInterface {
	result := make([]types.NetworkInterface, n)
	for i := 0; i < n; i++ {
		result[i] = types.NetworkInterface{
			NetworkInterfaceId: aws.String(fmt.Sprintf("eni-%04d", i)),
		}
	}
	return result
}

// TestGetNetworkInterfaces_Pagination verifies that GetNetworkInterfaces
// retrieves every ENI across multiple pages. Before the fix the helper only
// called DescribeNetworkInterfaces once, so accounts with more ENIs than a
// single page would see truncated output in `awstools vpc enis`.
func TestGetNetworkInterfaces_Pagination(t *testing.T) {
	total := 5
	mock := &mockDescribeNetworkInterfacesAPIClient{
		interfaces: makeNetworkInterfaces(total),
		pageSize:   2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := GetNetworkInterfaces(mock)

	if len(result) != total {
		t.Fatalf("GetNetworkInterfaces() returned %d ENIs, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.callCount != 3 {
		t.Errorf("DescribeNetworkInterfaces called %d times, want 3 (one per page)", mock.callCount)
	}
	for i, eni := range result {
		want := fmt.Sprintf("eni-%04d", i)
		if aws.ToString(eni.NetworkInterfaceId) != want {
			t.Errorf("GetNetworkInterfaces()[%d].NetworkInterfaceId = %s, want %s", i, aws.ToString(eni.NetworkInterfaceId), want)
		}
	}
}
