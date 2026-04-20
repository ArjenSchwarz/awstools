package helpers

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Regression tests for T-757: searchENIsByIP previously issued a single
// DescribeNetworkInterfaces call and ignored pagination. In accounts with many
// ENIs — including multi-VPC environments where the same RFC1918 address can
// appear on ENIs in several VPCs — matches on later pages were silently
// dropped, hiding multi-match warnings and returning incomplete results.

// makeNetworkInterfacesWithIP builds n synthetic ENIs whose primary private
// address is the supplied IP so filter-based searches can be simulated.
func makeNetworkInterfacesWithIP(n int, ip string, idPrefix string) []types.NetworkInterface {
	result := make([]types.NetworkInterface, n)
	for i := 0; i < n; i++ {
		result[i] = types.NetworkInterface{
			NetworkInterfaceId: aws.String(fmt.Sprintf("%s-%04d", idPrefix, i)),
			VpcId:              aws.String(fmt.Sprintf("vpc-%04d", i)),
			PrivateIpAddress:   aws.String(ip),
			PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
				{
					PrivateIpAddress: aws.String(ip),
					Primary:          aws.Bool(true),
				},
			},
		}
	}
	return result
}

// TestSearchENIsByIP_Pagination verifies that searchENIsByIP walks every page
// of DescribeNetworkInterfaces. Before the fix only the first page was
// processed, so matches on page 2+ were lost — including in multi-VPC
// environments where the filter `addresses.private-ip-address` could legitimately
// match ENIs across many VPCs that reuse the same private IP.
func TestSearchENIsByIP_Pagination(t *testing.T) {
	total := 5
	mock := &mockDescribeNetworkInterfacesAPIClient{
		interfaces: makeNetworkInterfacesWithIP(total, "10.0.1.100", "eni-match"),
		pageSize:   2, // force 3 pages: [0,1], [2,3], [4]
	}

	filters := []types.Filter{
		{
			Name:   aws.String("addresses.private-ip-address"),
			Values: []string{"10.0.1.100"},
		},
	}

	result := searchENIsByIP(mock, filters)

	if len(result) != total {
		t.Fatalf("searchENIsByIP() returned %d ENIs, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.callCount != 3 {
		t.Errorf("DescribeNetworkInterfaces called %d times, want 3 (one per page)", mock.callCount)
	}
	for i, eni := range result {
		want := fmt.Sprintf("eni-match-%04d", i)
		if aws.ToString(eni.NetworkInterfaceId) != want {
			t.Errorf("searchENIsByIP()[%d].NetworkInterfaceId = %s, want %s", i, aws.ToString(eni.NetworkInterfaceId), want)
		}
	}
}

// TestSearchENIsByIP_PaginationMultiVPC simulates the exact scenario from the
// ticket: the same private IP appears on ENIs across multiple VPCs spread over
// several response pages. The helper must return every match so the caller can
// emit the "multiple matches" warning instead of silently returning the first
// page's entry.
func TestSearchENIsByIP_PaginationMultiVPC(t *testing.T) {
	// 4 ENIs, each in a different VPC, all with the same IP
	interfaces := makeNetworkInterfacesWithIP(4, "10.0.1.100", "eni-vpc")
	mock := &mockDescribeNetworkInterfacesAPIClient{
		interfaces: interfaces,
		pageSize:   1, // force 4 pages, one ENI per page
	}

	filters := []types.Filter{
		{
			Name:   aws.String("addresses.private-ip-address"),
			Values: []string{"10.0.1.100"},
		},
	}

	result := searchENIsByIP(mock, filters)

	if len(result) != 4 {
		t.Fatalf("searchENIsByIP() returned %d ENIs across VPCs, want 4 (later-page matches dropped)", len(result))
	}
	if mock.callCount != 4 {
		t.Errorf("DescribeNetworkInterfaces called %d times, want 4 (one per page)", mock.callCount)
	}

	// Ensure each result comes from a distinct VPC — proving we collected
	// matches across pages rather than returning duplicates from page 1.
	seen := make(map[string]bool)
	for _, eni := range result {
		vpcID := aws.ToString(eni.VpcId)
		if seen[vpcID] {
			t.Errorf("duplicate VPC %s in results — paginator may be re-reading page 1", vpcID)
		}
		seen[vpcID] = true
	}
}

// TestSearchENIsByIP_SinglePage verifies the helper still works when results
// fit in a single page — no unnecessary additional calls.
func TestSearchENIsByIP_SinglePage(t *testing.T) {
	mock := &mockDescribeNetworkInterfacesAPIClient{
		interfaces: makeNetworkInterfacesWithIP(2, "10.0.1.100", "eni-single"),
		// pageSize 0 → everything in one page
	}

	filters := []types.Filter{
		{
			Name:   aws.String("addresses.private-ip-address"),
			Values: []string{"10.0.1.100"},
		},
	}

	result := searchENIsByIP(mock, filters)

	if len(result) != 2 {
		t.Fatalf("searchENIsByIP() returned %d ENIs, want 2", len(result))
	}
	if mock.callCount != 1 {
		t.Errorf("DescribeNetworkInterfaces called %d times, want 1", mock.callCount)
	}
}

// TestSearchENIsByIP_NoMatches verifies the helper returns an empty slice when
// no ENIs match the filter.
func TestSearchENIsByIP_NoMatches(t *testing.T) {
	mock := &mockDescribeNetworkInterfacesAPIClient{
		interfaces: []types.NetworkInterface{},
	}

	filters := []types.Filter{
		{
			Name:   aws.String("addresses.private-ip-address"),
			Values: []string{"10.0.1.100"},
		},
	}

	result := searchENIsByIP(mock, filters)

	if len(result) != 0 {
		t.Errorf("searchENIsByIP() returned %d ENIs, want 0", len(result))
	}
	if mock.callCount != 1 {
		t.Errorf("DescribeNetworkInterfaces called %d times, want 1", mock.callCount)
	}
}
